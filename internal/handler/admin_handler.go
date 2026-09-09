// Package handler — administration d'ABMCY Core : gestion des applications
// clientes (apps) qui intègrent le widget/SDK de paiement. Même principe que
// GatewayHandler côté DIARRA (backend/internal/handler/admin_handler.go,
// ListGatewayClients/CreateGatewayClient/SetGatewayClientActive), un niveau
// au-dessus : ici on gère les CLIENTS d'ABMCY Core, pas ceux de DIARRA.
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/abmcy/core/internal/auth"
	"github.com/abmcy/core/internal/model"
	"github.com/abmcy/core/internal/repository"
	"github.com/go-chi/chi/v5"
)

// relayer : ce que l'admin peut redéclencher (implémenté par PaymentHandler).
type relayer interface {
	RelayPayment(ctx context.Context, p *model.Payment) error
	RefundDeposit(ctx context.Context, deposit *model.Payment) (*model.Payment, error)
	PayoutForAdmin(ctx context.Context, p PayoutAdminParams) (*model.Payment, error)
}

type AdminHandler struct {
	appRepo *repository.AppRepo
	relay   relayer
}

func NewAdminHandler(appRepo *repository.AppRepo, relay relayer) *AdminHandler {
	return &AdminHandler{appRepo: appRepo, relay: relay}
}

// Dashboard — GET /admin/stats?since=7|30|0 : chiffres agrégés.
func (h *AdminHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	since := 30
	if v := r.URL.Query().Get("since"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			since = n
		}
	}
	d, err := h.appRepo.Dashboard(r.Context(), since)
	if err != nil {
		http.Error(w, `{"error":"stats_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(d)
}

// ListApps — GET /admin/apps
func (h *AdminHandler) ListApps(w http.ResponseWriter, r *http.Request) {
	apps, err := h.appRepo.List(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"apps": apps})
}

// CreateApp — POST /admin/apps. Génère une clé API + un secret HMAC opaques
// (jamais stockés en clair, voir auth.HashToken) et les renvoie UNE SEULE
// FOIS dans la réponse — exactement le même principe que
// AdminHandler.CreateGatewayClient côté DIARRA.
func (h *AdminHandler) CreateApp(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name               string `json:"name"`
		DefaultCallbackURL string `json:"default_callback_url,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if input.Name == "" {
		http.Error(w, `{"error":"name_required"}`, http.StatusBadRequest)
		return
	}

	apiKey, err := auth.GenerateToken()
	if err != nil {
		http.Error(w, `{"error":"key_generation_failed"}`, http.StatusInternalServerError)
		return
	}
	hmacSecret, err := auth.GenerateToken()
	if err != nil {
		http.Error(w, `{"error":"key_generation_failed"}`, http.StatusInternalServerError)
		return
	}

	var callbackURL *string
	if input.DefaultCallbackURL != "" {
		callbackURL = &input.DefaultCallbackURL
	}

	app, err := h.appRepo.Create(r.Context(), input.Name, auth.HashToken(apiKey), auth.HashToken(hmacSecret), callbackURL)
	if err != nil {
		http.Error(w, `{"error":"creation_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"app":         app,
		"api_key":     apiKey,     // affichée une seule fois — à transmettre à l'app cliente maintenant
		"hmac_secret": hmacSecret, // idem : ni l'une ni l'autre ne seront jamais revisibles après cette réponse
	})
}

// ListPayments — GET /admin/payments?app_id=&status=&limit=&offset=
func (h *AdminHandler) ListPayments(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	payments, err := h.appRepo.ListPayments(r.Context(), repository.ListPaymentsFilter{
		AppID:  q.Get("app_id"),
		Status: q.Get("status"),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"payments": payments})
}

// GetPayment — GET /admin/payments/{id}
func (h *AdminHandler) GetPayment(w http.ResponseWriter, r *http.Request) {
	p, err := h.appRepo.FindPaymentByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"payment": p})
}

// RelayPayment — POST /admin/payments/{id}/relay : redéclenche à la main le
// relais vers l'app cliente (callback raté, app remise en ligne...). Réponse
// synchrone avec l'issue, pour que le dashboard l'affiche tout de suite.
func (h *AdminHandler) RelayPayment(w http.ResponseWriter, r *http.Request) {
	p, err := h.appRepo.FindPaymentByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	relayErr := h.relay.RelayPayment(r.Context(), p)
	// Recharge pour renvoyer relay_status / relay_last_error à jour.
	fresh, _ := h.appRepo.FindPaymentByID(r.Context(), p.ID)
	w.Header().Set("Content-Type", "application/json")
	// Toujours 200 : un échec de relais est un résultat métier normal (l'app
	// est peut-être hors ligne), pas une erreur de CETTE requête. Renvoyer un
	// 5xx ici ferait aussi intercepter la réponse par Cloudflare (page 502
	// générique au lieu de notre JSON). Le dashboard lit `ok`.
	if relayErr != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": relayErr.Error(), "payment": fresh})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "payment": fresh})
}

// CreatePayout — POST /admin/payouts : déclenche un versement depuis la
// console. Corps : {app_id, amount_cfa, recipient_phone, recipient_operator,
// country, description?}. Toujours 200, {ok, error?, payout?}.
func (h *AdminHandler) CreatePayout(w http.ResponseWriter, r *http.Request) {
	var in struct {
		AppID             string `json:"app_id"`
		AmountCFA         int    `json:"amount_cfa"`
		RecipientPhone    string `json:"recipient_phone"`
		RecipientOperator string `json:"recipient_operator"`
		Country           string `json:"country"`
		Description       string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if in.AppID == "" {
		http.Error(w, `{"error":"app_id_required"}`, http.StatusBadRequest)
		return
	}
	payout, err := h.relay.PayoutForAdmin(r.Context(), PayoutAdminParams{
		AppID:             in.AppID,
		AmountCFA:         in.AmountCFA,
		RecipientPhone:    in.RecipientPhone,
		RecipientOperator: in.RecipientOperator,
		Country:           in.Country,
		Description:       in.Description,
	})
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "payout": payout})
}

// RefundPayment — POST /admin/payments/{id}/refund : rembourse à la main un
// dépôt "completed". Toujours 200, {ok, error?, refund?}.
func (h *AdminHandler) RefundPayment(w http.ResponseWriter, r *http.Request) {
	deposit, err := h.appRepo.FindPaymentByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	refund, refErr := h.relay.RefundDeposit(r.Context(), deposit)
	w.Header().Set("Content-Type", "application/json")
	if refErr != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": refErr.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "refund": refund})
}

// SetAppKYC — PUT /admin/apps/{id}/kyc. Corps : {"level": "none"|"verified"}.
// Validé à la main après réception des justificatifs. "verified" fait passer
// le plafond par transaction de 200 000 à 1 000 000 FCFA (voir model.MaxAmountFor).
func (h *AdminHandler) SetAppKYC(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var input struct {
		Level string `json:"level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if input.Level != model.KYCNone && input.Level != model.KYCVerified {
		http.Error(w, `{"error":"level_must_be_none_or_verified"}`, http.StatusBadRequest)
		return
	}
	if err := h.appRepo.SetKYCLevel(r.Context(), id, input.Level); err != nil {
		http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// SetAppActive — PUT /admin/apps/{id}/active. Corps : {"active": true|false}.
func (h *AdminHandler) SetAppActive(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var input struct {
		Active bool `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if err := h.appRepo.SetActive(r.Context(), id, input.Active); err != nil {
		http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}
