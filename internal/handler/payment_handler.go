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
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

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
}

func NewPaymentHandler(appRepo *repository.AppRepo, diarra *gateway.Client, diarraHMACSecretHash, selfCallbackURL string) *PaymentHandler {
	return &PaymentHandler{
		appRepo:              appRepo,
		diarra:               diarra,
		diarraHMACSecretHash: diarraHMACSecretHash,
		selfCallbackURL:      selfCallbackURL,
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
		writePayment(w, existing, http.StatusOK)
		return
	}

	diarraRef := abmcyUUID()
	var desc, callbackURL *string
	if input.Description != "" {
		desc = &input.Description
	}
	if input.CallbackURL != "" {
		callbackURL = &input.CallbackURL
	}

	payment, err := h.appRepo.CreatePayment(r.Context(), repository.CreatePaymentParams{
		AppID:           appID,
		AppRef:          input.AppRef,
		DiarraClientRef: diarraRef,
		AmountCFA:       input.AmountCFA,
		Currency:        "XOF",
		Description:     desc,
		CallbackURL:     callbackURL,
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
	writePayment(w, payment, http.StatusCreated)
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

	go h.relayToApp(payment)

	w.WriteHeader(http.StatusOK)
}

// relayToApp notifie l'app cliente (POST callback_url) qu'un paiement a
// changé de statut — best-effort, en tâche de fond, même principe que
// WebhookHandler.relayGatewayCallback côté DIARRA (jamais bloquant, un
// callback manqué se rattrape via GET /v1/payments/{app_ref}).
func (h *PaymentHandler) relayToApp(p *model.Payment) {
	if p.CallbackURL == nil || *p.CallbackURL == "" {
		return
	}
	body, err := json.Marshal(map[string]interface{}{
		"app_ref":        p.AppRef,
		"status":         p.Status,
		"amount_cfa":     p.AmountCFA,
		"failure_reason": p.FailureReason,
	})
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, *p.CallbackURL, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("relais app: callback %s injoignable pour payment=%s: %v", *p.CallbackURL, p.ID, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("relais app: callback %s a répondu %d pour payment=%s", *p.CallbackURL, resp.StatusCode, p.ID)
	}
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
