// Package handler — inscription publique (landing page) et son back-office.
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"

	"github.com/abmcy/core/internal/auth"
	"github.com/abmcy/core/internal/model"
	"github.com/abmcy/core/internal/repository"
	"github.com/go-chi/chi/v5"
)

type SignupHandler struct {
	signups *repository.SignupRepo
	apps    *repository.AppRepo
}

func NewSignupHandler(signups *repository.SignupRepo, apps *repository.AppRepo) *SignupHandler {
	return &SignupHandler{signups: signups, apps: apps}
}

// PublicSignup — POST /public/signup (aucune auth) : le formulaire de la
// landing page. Crée une demande 'pending'. Ne révèle jamais si l'email a
// déjà une app — juste "déjà une demande en attente" le cas échéant.
func (h *SignupHandler) PublicSignup(w http.ResponseWriter, r *http.Request) {
	var in model.PublicSignupInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&in); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "invalid_request")
		return
	}
	in.BusinessName = strings.TrimSpace(in.BusinessName)
	in.ContactName = strings.TrimSpace(in.ContactName)
	in.Email = strings.TrimSpace(in.Email)

	if in.BusinessName == "" || in.ContactName == "" || in.Email == "" {
		writeJSONErr(w, http.StatusBadRequest, "business_name, contact_name et email sont requis")
		return
	}
	if _, err := mail.ParseAddress(in.Email); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "email invalide")
		return
	}

	req, err := h.signups.Create(r.Context(), in)
	if err != nil {
		if errors.Is(err, repository.ErrSignupPendingExists) {
			writeJSONErr(w, http.StatusConflict, "Une demande est déjà en cours pour cet email. Nous revenons vers vous rapidement.")
			return
		}
		writeJSONErr(w, http.StatusInternalServerError, "creation_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"ok":      true,
		"id":      req.ID,
		"message": "Demande reçue. Vous recevrez vos accès par email après validation.",
	})
}

// --- Back-office (Bearer ABMCY_ADMIN_TOKEN) ---------------------------

// ListSignups — GET /admin/signups?status=pending|approved|rejected
func (h *SignupHandler) ListSignups(w http.ResponseWriter, r *http.Request) {
	reqs, err := h.signups.List(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "list_failed")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"signups": reqs})
}

// ApproveSignup — POST /admin/signups/{id}/approve. Corps optionnel
// {"note": "..."}. Crée l'app, lie la demande, et renvoie la clé API + le
// secret HMAC UNE SEULE FOIS (l'admin les transmet au demandeur par email).
func (h *SignupHandler) ApproveSignup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Note string `json:"note"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	req, err := h.signups.FindByID(r.Context(), id)
	if err != nil {
		writeJSONErr(w, http.StatusNotFound, "not_found")
		return
	}
	if req.Status != model.SignupPending {
		writeJSONErr(w, http.StatusConflict, "demande déjà traitée")
		return
	}

	apiKey, err := auth.GenerateToken()
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "key_generation_failed")
		return
	}
	hmacSecret, err := auth.GenerateToken()
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "key_generation_failed")
		return
	}

	// Pas de callback_url par défaut : rien de fiable à deviner depuis le
	// site web, l'app le configurera elle-même.
	app, err := h.apps.Create(r.Context(), req.BusinessName, auth.HashToken(apiKey), auth.HashToken(hmacSecret), nil)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "app_creation_failed")
		return
	}
	if err := h.signups.MarkApproved(r.Context(), id, app.ID, body.Note); err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "update_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"ok":          true,
		"app":         app,
		"api_key":     apiKey,
		"hmac_secret": hmacSecret,
		"email":       req.Email,
		"note":        "Transmettez ces identifiants au demandeur — ils ne seront plus jamais affichés.",
	})
}

// RejectSignup — POST /admin/signups/{id}/reject. Corps {"note": "motif"}.
func (h *SignupHandler) RejectSignup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Note string `json:"note"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	req, err := h.signups.FindByID(r.Context(), id)
	if err != nil {
		writeJSONErr(w, http.StatusNotFound, "not_found")
		return
	}
	if req.Status != model.SignupPending {
		writeJSONErr(w, http.StatusConflict, "demande déjà traitée")
		return
	}
	if err := h.signups.MarkRejected(r.Context(), id, body.Note); err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "update_failed")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func writeJSONErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
