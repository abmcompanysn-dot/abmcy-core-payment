// Package handler — endpoints HTTP d'ABMCY Core pour le flux de paiement.
// ABMCY Core est un CLIENT de la passerelle DIARRA (voir internal/gateway) :
// il ne parle jamais directement à PawaPay/KPay/PayPal.
package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/abmcy/core/internal/gateway"
	"github.com/go-chi/chi/v5"
)

type PaymentHandler struct {
	diarra         *gateway.Client
	hmacSecretHash string // même valeur que gateway.Config.HMACSecretHash — vérifie les callbacks REÇUS de DIARRA
}

func NewPaymentHandler(diarra *gateway.Client, hmacSecretHash string) *PaymentHandler {
	return &PaymentHandler{diarra: diarra, hmacSecretHash: hmacSecretHash}
}

// Pay — POST /pay {"client_ref": "...", "amount_cfa": 1000, "country": "SEN"}
// Ouvre un dépôt via la passerelle DIARRA et renvoie l'URL de paiement
// hébergée (PawaPay) vers laquelle rediriger l'utilisateur final d'ABMCY Core.
func (h *PaymentHandler) Pay(w http.ResponseWriter, r *http.Request) {
	var input gateway.CreateDepositInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if input.ClientRef == "" || input.AmountCFA <= 0 || input.Country == "" {
		http.Error(w, `{"error":"client_ref_amount_country_required"}`, http.StatusBadRequest)
		return
	}

	tx, err := h.diarra.CreateDeposit(input)
	if err != nil {
		log.Printf("pay: échec création dépôt via DIARRA: %v", err)
		http.Error(w, `{"error":"payment_init_failed"}`, http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"client_ref":   tx.ClientRef,
		"status":       tx.Status,
		"redirect_url": tx.RedirectURL,
	})
}

// PaymentStatus — GET /payments/{client_ref} : relit le statut auprès de
// DIARRA (au cas où le callback ne serait pas encore arrivé, ou aurait été
// manqué) plutôt que de ne se fier qu'au cache local d'ABMCY Core.
func (h *PaymentHandler) PaymentStatus(w http.ResponseWriter, r *http.Request) {
	clientRef := chi.URLParam(r, "client_ref")
	tx, err := h.diarra.GetDeposit(clientRef)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tx)
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
// d'une transaction (dépôt/versement/remboursement) initiée via CreateDeposit/
// CreatePayout/CreateRefund. Vérifie X-Diarra-Gateway-Signature avant de
// traiter quoi que ce soit — jamais confiance dans un corps non signé.
func (h *PaymentHandler) DiarraCallback(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"read_failed"}`, http.StatusBadRequest)
		return
	}
	sig := r.Header.Get("X-Diarra-Gateway-Signature")
	if !verifySignature(h.hmacSecretHash, body, sig) {
		http.Error(w, `{"error":"invalid_signature"}`, http.StatusUnauthorized)
		return
	}

	var payload gatewayCallbackPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	// TODO produit réel : mettre à jour l'état métier d'ABMCY Core pour
	// payload.ClientRef ici (ex. marquer une commande payée, créditer un
	// solde...). Ce squelette se contente de tracer l'événement pour
	// prouver le bout-en-bout DIARRA -> ABMCY Core.
	log.Printf("callback DIARRA reçu: client_ref=%s type=%s status=%s montant=%d failure=%q",
		payload.ClientRef, payload.Type, payload.Status, payload.AmountCFA, payload.FailureReason)

	w.WriteHeader(http.StatusOK)
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
