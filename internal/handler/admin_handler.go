// Package handler — administration d'ABMCY Core : gestion des applications
// clientes (apps) qui intègrent le widget/SDK de paiement. Même principe que
// GatewayHandler côté DIARRA (backend/internal/handler/admin_handler.go,
// ListGatewayClients/CreateGatewayClient/SetGatewayClientActive), un niveau
// au-dessus : ici on gère les CLIENTS d'ABMCY Core, pas ceux de DIARRA.
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/abmcy/core/internal/auth"
	"github.com/abmcy/core/internal/repository"
	"github.com/go-chi/chi/v5"
)

type AdminHandler struct {
	appRepo *repository.AppRepo
}

func NewAdminHandler(appRepo *repository.AppRepo) *AdminHandler {
	return &AdminHandler{appRepo: appRepo}
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
