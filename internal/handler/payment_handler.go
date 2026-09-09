// Package handler — endpoints HTTP d'ABMCY Core pour le flux de paiement.
// ABMCY Core est un ORCHESTRATEUR à deux niveaux :
//
//	app cliente (widget/SDK) -> ABMCY Core -> passerelle DIARRA -> agrégateur
//
// Il ne parle jamais directement à PawaPay/KPay/PayPal (voir internal/gateway,
// seul point de contact avec DIARRA) ; chaque niveau a sa propre paire
// clé API/secret HMAC (voir internal/middleware/app_auth.go pour ce niveau-ci).
package handler

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/abmcy/core/internal/fees"
	"github.com/abmcy/core/internal/gateway"
	"github.com/abmcy/core/internal/middleware"
	"github.com/abmcy/core/internal/model"
	"github.com/abmcy/core/internal/repository"
	"github.com/go-chi/chi/v5"
)

type PaymentHandler struct {
	appRepo *repository.AppRepo
	diarra  *gateway.Client
	// diarraHMACSecretHash : vérifie les callbacks REÇUS de DIARRA
	// (X-Diarra-Gateway-Signature) — même valeur que gateway.Config.HMACSecretHash,
	// UNE SEULE paire clé/secret pour toute l'instance ABMCY Core côté DIARRA
	// (contrairement aux apps clientes, qui ont chacune la leur).
	diarraHMACSecretHash string
	// selfCallbackURL : l'URL PUBLIQUE de CE service que DIARRA doit rappeler
	// (ex. https://core.diarra.app/webhooks/diarra). Obligatoire : DIARRA ne
	// relaie un callback QUE si la transaction porte un callback_url explicite
	// (relayGatewayCallback abandonne sinon, pas de fallback sur
	// default_callback_url), donc on le passe sur CHAQUE CreateDeposit.
	selfCallbackURL string
	// publicBaseURL : racine publique de CE service (ex. https://core.diarra.app),
	// dérivée de selfCallbackURL. Sert à construire hosted_pay_url renvoyée à
	// l'app (page de paiement hébergée /pay/{ref}).
	publicBaseURL string
	// usdRate : taux FCFA/USD pour le calcul de commission (voir fees.Compute).
	usdRate int
}

func NewPaymentHandler(appRepo *repository.AppRepo, diarra *gateway.Client, diarraHMACSecretHash, selfCallbackURL string, usdRate int) *PaymentHandler {
	if usdRate <= 0 {
		usdRate = fees.DefaultUSDRate
	}
	return &PaymentHandler{
		appRepo:              appRepo,
		diarra:               diarra,
		diarraHMACSecretHash: diarraHMACSecretHash,
		selfCallbackURL:      selfCallbackURL,
		publicBaseURL:        strings.TrimSuffix(strings.TrimSuffix(selfCallbackURL, "/webhooks/diarra"), "/"),
		usdRate:              usdRate,
	}
}

// LookupApp — adapte AppRepo au type attendu par middleware.RequireApp.
func (h *PaymentHandler) LookupApp(ctx context.Context, apiKeyHash string) (middleware.AppLookup, error) {
	a, err := h.appRepo.FindByKeyHash(ctx, apiKeyHash)
	if err != nil {
		return middleware.AppLookup{}, err
	}
	return middleware.AppLookup{ID: a.ID, HMACSecretHash: a.HMACSecretHash}, nil
}

func abmcyUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// Pay — POST /v1/pay (authentifié par app, voir middleware.RequireApp).
// Crée un enregistrement Payment côté ABMCY Core, PUIS un dépôt via la
// passerelle DIARRA avec un diarra_client_ref généré ici (jamais l'app_ref de
// l'app cliente — isolation : DIARRA ne connaît qu'ABMCY Core comme client,
// jamais l'app finale). Renvoie l'URL de paiement hébergée à ouvrir pour
// l'utilisateur final.
func (h *PaymentHandler) Pay(w http.ResponseWriter, r *http.Request) {
	appID := middleware.GetAppID(r.Context())

	var input model.CreatePaymentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if input.AppRef == "" || input.AmountCFA <= 0 || input.Country == "" {
		http.Error(w, `{"error":"app_ref_amount_country_required"}`, http.StatusBadRequest)
		return
	}

	// Idempotence : un app_ref déjà utilisé par CETTE app renvoie le paiement
	// existant plutôt que d'en créer un second (même principe que
	// GatewayHandler.CreateDeposit côté DIARRA).
	if existing, err := h.appRepo.FindPaymentByAppRef(r.Context(), appID, input.AppRef); err == nil {
		writePaymentWithHosted(w, existing, h.hostedPayURL(existing.DiarraClientRef), http.StatusOK)
		return
	}

	// Plafond par transaction selon le niveau KYC de l'app. Sans KYC :
	// 200 000 FCFA ; vérifiée : 1 000 000. Au-delà, on refuse net.
	app, err := h.appRepo.FindByID(r.Context(), appID)
	if err != nil {
		http.Error(w, `{"error":"app_lookup_failed"}`, http.StatusInternalServerError)
		return
	}
	maxAmt := model.MaxAmountFor(app.KYCLevel)
	if input.AmountCFA > maxAmt {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":        "limit_exceeded",
			"message":      fmt.Sprintf("Montant maximum autorisé : %d FCFA par transaction. Vérification d'identité requise pour aller au-delà.", maxAmt),
			"max_cfa":      maxAmt,
			"kyc_level":    app.KYCLevel,
			"kyc_required": app.KYCLevel == model.KYCNone,
		})
		return
	}

	// Commission ABMCY Core, figée maintenant (le taux peut bouger ensuite).
	fee := fees.Compute(input.AmountCFA, h.usdRate)

	diarraRef := abmcyUUID()
	var desc, callbackURL, returnURL *string
	if input.Description != "" {
		desc = &input.Description
	}
	if input.CallbackURL != "" {
		callbackURL = &input.CallbackURL
	}
	if input.ReturnURL != "" {
		returnURL = &input.ReturnURL
	}

	payment, err := h.appRepo.CreatePayment(r.Context(), repository.CreatePaymentParams{
		AppID:           appID,
		AppRef:          input.AppRef,
		DiarraClientRef: diarraRef,
		AmountCFA:       input.AmountCFA,
		FeeCFA:          fee.FeeCFA,
		NetCFA:          fee.NetCFA,
		USDRateUsed:     fee.USDRateUsed,
		Currency:        "XOF",
		Description:     desc,
		CallbackURL:     callbackURL,
		ReturnURL:       returnURL,
	})
	if err != nil {
		http.Error(w, `{"error":"payment_creation_failed"}`, http.StatusInternalServerError)
		return
	}

	// ABMCY Core reçoit lui-même le callback DIARRA sur son URL fixe
	// (/webhooks/diarra) — jamais l'URL de l'app cliente directement, pour
	// garder la main sur la mise à jour de `payments` avant de relayer.
	tx, err := h.diarra.CreateDeposit(gateway.CreateDepositInput{
		ClientRef:   diarraRef,
		AmountCFA:   input.AmountCFA,
		Country:     input.Country,
		Description: input.Description,
		CallbackURL: h.selfCallbackURL, // DIARRA -> /webhooks/diarra de CE service (jamais l'URL de l'app)
	})
	if err != nil {
		log.Printf("pay: échec création dépôt via DIARRA pour payment=%s: %v", payment.ID, err)
		reason := "diarra_init_failed"
		_ = h.appRepo.UpdatePaymentStatus(r.Context(), payment.ID, model.PaymentFailed, nil, &reason)
		http.Error(w, `{"error":"payment_init_failed"}`, http.StatusBadGateway)
		return
	}
	if err := h.appRepo.UpdatePaymentRedirect(r.Context(), payment.ID, tx.RedirectURL); err != nil {
		http.Error(w, `{"error":"payment_update_failed"}`, http.StatusInternalServerError)
		return
	}

	payment.RedirectURL = &tx.RedirectURL
	writePaymentWithHosted(w, payment, h.hostedPayURL(payment.DiarraClientRef), http.StatusCreated)
}

// hostedPayURL construit l'URL de la page de paiement hébergée d'ABMCY Core
// pour ce paiement (l'app ouvre ce lien pour l'utilisateur au lieu de gérer
// la redirection PawaPay elle-même).
func (h *PaymentHandler) hostedPayURL(diarraRef string) string {
	if h.publicBaseURL == "" {
		return ""
	}
	return h.publicBaseURL + "/pay/" + diarraRef
}

// PaymentStatus — GET /v1/payments/{app_ref} (authentifié par app) : lit
// l'état local, déjà tenu à jour par les callbacks DIARRA (voir
// DiarraCallback) — pas d'appel réseau supplémentaire nécessaire.
func (h *PaymentHandler) PaymentStatus(w http.ResponseWriter, r *http.Request) {
	appID := middleware.GetAppID(r.Context())
	appRef := chi.URLParam(r, "app_ref")
	payment, err := h.appRepo.FindPaymentByAppRef(r.Context(), appID, appRef)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	writePayment(w, payment, http.StatusOK)
}

func writePayment(w http.ResponseWriter, p *model.Payment, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{"payment": p})
}

func writePaymentWithHosted(w http.ResponseWriter, p *model.Payment, hostedPayURL string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	out := map[string]interface{}{"payment": p}
	if hostedPayURL != "" {
		out["hosted_pay_url"] = hostedPayURL
	}
	json.NewEncoder(w).Encode(out)
}

// gatewayCallbackPayload — même forme que côté DIARRA (gateway_relay.go),
// dupliquée ici volontairement : ABMCY Core ne doit dépendre d'aucun package
// interne DIARRA, seulement du contrat HTTP public de la passerelle.
type gatewayCallbackPayload struct {
	ClientRef     string `json:"client_ref"`
	Type          string `json:"type"`
	Status        string `json:"status"`
	FailureReason string `json:"failure_reason,omitempty"`
	AmountCFA     int    `json:"amount_cfa"`
}

// DiarraCallback — POST /webhooks/diarra : DIARRA relaie ici le statut final
// d'un dépôt/versement/remboursement (client_ref = diarra_client_ref généré
// par Pay ci-dessus). Vérifie X-Diarra-Gateway-Signature avant de traiter
// quoi que ce soit, met à jour le Payment correspondant, PUIS relaie vers
// l'app cliente d'origine (son callback_url, jamais celui de DIARRA) —
// exactement le même principe de relais que DIARRA applique déjà pour ses
// propres callbacks agrégateur.
func (h *PaymentHandler) DiarraCallback(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"read_failed"}`, http.StatusBadRequest)
		return
	}
	sig := r.Header.Get("X-Diarra-Gateway-Signature")
	if !verifySignature(h.diarraHMACSecretHash, body, sig) {
		http.Error(w, `{"error":"invalid_signature"}`, http.StatusUnauthorized)
		return
	}

	var payload gatewayCallbackPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	payment, err := h.appRepo.FindPaymentByDiarraRef(r.Context(), payload.ClientRef)
	if err != nil {
		log.Printf("callback DIARRA: aucun paiement pour diarra_client_ref=%s", payload.ClientRef)
		// 200 quand même : DIARRA ne doit jamais re-livrer indéfiniment un
		// callback qu'on ne saura de toute façon jamais rattacher.
		w.WriteHeader(http.StatusOK)
		return
	}

	var failureReason *string
	if payload.FailureReason != "" {
		failureReason = &payload.FailureReason
	}
	if err := h.appRepo.UpdatePaymentStatus(r.Context(), payment.ID, payload.Status, nil, failureReason); err != nil {
		log.Printf("callback DIARRA: échec mise à jour payment=%s: %v", payment.ID, err)
		http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
		return
	}
	payment.Status = payload.Status
	payment.FailureReason = failureReason

	go func() {
		if err := h.RelayPayment(context.Background(), payment); err != nil {
			log.Printf("relais app: payment=%s: %v", payment.ID, err)
		}
	}()

	w.WriteHeader(http.StatusOK)
}

// RelayPayment notifie l'app cliente (POST callback_url signé) qu'un
// paiement a changé de statut, et enregistre l'issue (relay_status) pour le
// dashboard admin. Best-effort au sens où un échec n'annule rien côté ABMCY
// Core (l'app se resynchronise via GET /v1/payments/{app_ref} ou un renvoi
// manuel), mais l'issue est toujours tracée — jamais silencieuse. Réutilisé
// tel quel par AdminHandler pour le bouton "renvoyer le relais".
func (h *PaymentHandler) RelayPayment(ctx context.Context, p *model.Payment) error {
	if p.CallbackURL == nil || *p.CallbackURL == "" {
		return h.appRepo.SetRelayResult(ctx, p.ID, model.RelaySkipped, nil)
	}

	// Signature identique à celle que DIARRA nous envoie (HMAC du corps brut,
	// clé = hash du secret) mais avec le secret HMAC DE L'APP : l'app vérifie
	// que le callback vient bien d'ABMCY Core (voir middleware.RequireApp
	// côté vérif des requêtes entrantes de l'app).
	app, err := h.appRepo.FindByID(ctx, p.AppID)
	if err != nil {
		msg := "app introuvable pour la signature du relais"
		_ = h.appRepo.SetRelayResult(ctx, p.ID, model.RelayFailed, &msg)
		return err
	}

	body, err := json.Marshal(map[string]interface{}{
		"app_ref":        p.AppRef,
		"status":         p.Status,
		"amount_cfa":     p.AmountCFA,
		"failure_reason": p.FailureReason,
	})
	if err != nil {
		return err
	}

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, *p.CallbackURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	mac := hmac.New(sha256.New, []byte(app.HMACSecretHash))
	mac.Write(body)
	req.Header.Set("X-Abmcy-Signature", hex.EncodeToString(mac.Sum(nil)))

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		msg := err.Error()
		_ = h.appRepo.SetRelayResult(ctx, p.ID, model.RelayFailed, &msg)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		msg := fmt.Sprintf("callback a répondu HTTP %d", resp.StatusCode)
		_ = h.appRepo.SetRelayResult(ctx, p.ID, model.RelayFailed, &msg)
		return errors.New(msg)
	}
	return h.appRepo.SetRelayResult(ctx, p.ID, model.RelayDelivered, nil)
}

func verifySignature(hmacSecretHash string, body []byte, sigHex string) bool {
	if hmacSecretHash == "" || sigHex == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(hmacSecretHash))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(sigHex))
}

// --- Page de paiement hébergée -----------------------------------------

var hostedPayTmpl = template.Must(template.New("pay").Parse(`<!doctype html>
<html lang="fr"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Paiement — {{.AppName}}</title>
<style>
:root{color-scheme:light}
*{box-sizing:border-box}
body{margin:0;font:15px/1.5 -apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:#f4f5f7;color:#1a1a1a;display:flex;min-height:100vh;align-items:center;justify-content:center;padding:20px}
.card{background:#fff;border:1px solid #e3e5e8;border-radius:14px;max-width:420px;width:100%;padding:28px;box-shadow:0 2px 12px rgba(0,0,0,.06)}
h1{font-size:16px;margin:0 0 2px}
.merchant{color:#6b7280;font-size:13px;margin-bottom:20px}
.amount{font-size:32px;font-weight:700;letter-spacing:-.02em}
.desc{color:#4b5563;margin:6px 0 22px}
.st{display:inline-block;padding:3px 10px;border-radius:999px;font-size:12px;font-weight:700;text-transform:uppercase}
.st-pending{background:#fef3c7;color:#92400e}
.st-completed{background:#dcfce7;color:#166534}
.st-failed,.st-cancelled{background:#fee2e2;color:#991b1b}
button,a.btn{display:block;width:100%;text-align:center;background:#2f6bff;color:#fff;border:none;border-radius:9px;padding:13px;font-size:15px;font-weight:600;cursor:pointer;text-decoration:none;margin-top:8px}
a.btn.secondary{background:#fff;color:#374151;border:1px solid #d1d5db}
.foot{margin-top:20px;text-align:center;color:#9ca3af;font-size:11px}
</style></head><body>
<div class="card">
  <h1>{{.AppName}}</h1>
  <div class="merchant">Paiement sécurisé via ABMCY Core</div>
  {{if .IsPayable}}
    <div class="amount">{{.AmountFmt}} FCFA</div>
    {{if .Description}}<div class="desc">{{.Description}}</div>{{else}}<div class="desc">Commande {{.AppRef}}</div>{{end}}
    <a class="btn" href="{{.RedirectURL}}">Payer maintenant</a>
    {{if .ReturnURL}}<a class="btn secondary" href="{{.ReturnURL}}">Annuler et revenir</a>{{end}}
  {{else}}
    <div class="amount">{{.AmountFmt}} FCFA</div>
    <p>Statut du paiement : <span class="st st-{{.Status}}">{{.StatusLabel}}</span></p>
    {{if .FailureReason}}<p class="desc">{{.FailureReason}}</p>{{end}}
    {{if .ReturnURL}}<a class="btn" href="{{.ReturnURL}}">Revenir à la boutique</a>{{end}}
  {{end}}
  <div class="foot">core.diarra.app</div>
</div>
<script>
// Si cette page a été ouverte par le widget (window.opener), on prévient
// l'app du résultat puis on ferme. Sur un statut final, on ferme aussi
// automatiquement au bout de 2s si personne ne l'a fermée.
(function () {
  var status = {{.Status}};
  if (window.opener && !window.opener.closed) {
    try { window.opener.postMessage({ abmcy_pay: true, status: status, ref: {{.AppRef}} }, '*'); } catch (e) {}
  }
  if (status === 'completed' || status === 'failed' || status === 'cancelled') {
    setTimeout(function () { if (window.opener && !window.opener.closed) window.close(); }, 2000);
  }
})();
</script>
</body></html>`))

type hostedPayView struct {
	AppName       string
	AppRef        string
	AmountFmt     string
	Description   string
	Status        string
	StatusLabel   string
	FailureReason string
	RedirectURL   string
	ReturnURL     string
	IsPayable     bool
}

// HostedPay — GET /pay/{ref} : page de paiement hébergée, ouverte par le
// NAVIGATEUR de l'utilisateur final (aucune auth). {ref} = diarra_client_ref
// renvoyé dans hosted_pay_url par /v1/pay. Si le paiement est encore
// payable, un bouton renvoie vers la page PawaPay ; sinon on affiche le
// statut final et un lien de retour vers la boutique (return_url).
func (h *PaymentHandler) HostedPay(w http.ResponseWriter, r *http.Request) {
	ref := chi.URLParam(r, "ref")
	pw, err := h.appRepo.FindPaymentByDiarraRefPublic(r.Context(), ref)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("<!doctype html><meta charset=utf-8><p style=\"font:16px sans-serif;text-align:center;margin-top:20vh\">Paiement introuvable.</p>"))
		return
	}
	p := pw.Payment

	// Un paiement déjà terminé côté navigateur : renvoyer directement à la
	// boutique s'il y a une return_url, plutôt que d'afficher une page morte.
	if p.Status == model.PaymentCompleted && p.ReturnURL != nil && *p.ReturnURL != "" {
		http.Redirect(w, r, appendQuery(*p.ReturnURL, "status", "completed", "ref", p.AppRef), http.StatusFound)
		return
	}

	view := hostedPayView{
		AppName:     pw.AppName,
		AppRef:      p.AppRef,
		AmountFmt:   groupThousands(p.AmountCFA),
		Status:      p.Status,
		StatusLabel: statusLabelFR(p.Status),
		IsPayable:   (p.Status == model.PaymentPending || p.Status == model.PaymentProcessing) && p.RedirectURL != nil && *p.RedirectURL != "",
	}
	if p.Description != nil {
		view.Description = *p.Description
	}
	if p.FailureReason != nil {
		view.FailureReason = *p.FailureReason
	}
	if p.RedirectURL != nil {
		view.RedirectURL = *p.RedirectURL
	}
	if p.ReturnURL != nil {
		view.ReturnURL = *p.ReturnURL
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := hostedPayTmpl.Execute(w, view); err != nil {
		log.Printf("hosted pay: rendu template: %v", err)
	}
}

func statusLabelFR(s string) string {
	switch s {
	case model.PaymentCompleted:
		return "Payé"
	case model.PaymentFailed:
		return "Échoué"
	case model.PaymentCancelled:
		return "Annulé"
	case model.PaymentProcessing:
		return "En cours"
	default:
		return "En attente"
	}
}

func groupThousands(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var out []byte
	pre := len(s) % 3
	if pre > 0 {
		out = append(out, s[:pre]...)
	}
	for i := pre; i < len(s); i += 3 {
		if len(out) > 0 {
			out = append(out, ' ')
		}
		out = append(out, s[i:i+3]...)
	}
	return string(out)
}

func appendQuery(base string, kv ...string) string {
	u, err := url.Parse(base)
	if err != nil {
		return base
	}
	q := u.Query()
	for i := 0; i+1 < len(kv); i += 2 {
		q.Set(kv[i], kv[i+1])
	}
	u.RawQuery = q.Encode()
	return u.String()
}
