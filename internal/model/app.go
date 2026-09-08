package model

import "time"

// App — une application ABMCY qui intègre le widget/SDK de paiement ABMCY
// Core. Une par produit (une boutique, une app de réservation...) — pas
// l'utilisateur final, l'APPLICATION cliente elle-même. Voir migration 001.
type App struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	APIKeyHash         string    `json:"-"`
	HMACSecretHash     string    `json:"-"`
	DefaultCallbackURL *string   `json:"default_callback_url,omitempty"`
	IsActive           bool      `json:"is_active"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

const (
	PaymentTypeDeposit = "deposit"
	PaymentTypePayout  = "payout"
	PaymentTypeRefund  = "refund"

	PaymentPending    = "pending"
	PaymentProcessing = "processing"
	PaymentCompleted  = "completed"
	PaymentFailed     = "failed"
	PaymentCancelled  = "cancelled"
)

// Payment — un paiement initié par une App via ABMCY Core. AppRef est
// l'identifiant CHEZ L'APP (sa propre commande) ; DiarraClientRef est
// l'identifiant utilisé par ABMCY Core pour parler à la passerelle DIARRA
// (voir gateway.CreateDepositInput.ClientRef côté client DIARRA).
type Payment struct {
	ID              string    `json:"id"`
	AppID           string    `json:"-"`
	AppRef          string    `json:"app_ref"`
	DiarraClientRef string    `json:"-"`
	Type            string    `json:"type"`
	Provider        *string   `json:"provider,omitempty"`
	Status          string    `json:"status"`
	FailureReason   *string   `json:"failure_reason,omitempty"`
	AmountCFA       int       `json:"amount_cfa"`
	Currency        string    `json:"currency"`
	Description     *string   `json:"description,omitempty"`
	RedirectURL     *string   `json:"redirect_url,omitempty"`
	CallbackURL     *string   `json:"-"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CreatePaymentInput — POST /v1/pay (une app initie un paiement).
type CreatePaymentInput struct {
	AppRef      string `json:"app_ref"`
	AmountCFA   int    `json:"amount_cfa"`
	Country     string `json:"country"`
	Description string `json:"description,omitempty"`
	CallbackURL string `json:"callback_url,omitempty"`
}
