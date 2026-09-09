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
const (
	RelayPending   = "pending"
	RelayDelivered = "delivered"
	RelayFailed    = "failed"
	RelaySkipped   = "skipped" // pas de callback_url : rien à relayer
)

type Payment struct {
	ID              string  `json:"id"`
	AppID           string  `json:"-"`
	AppRef          string  `json:"app_ref"`
	DiarraClientRef string  `json:"-"`
	Type            string  `json:"type"`
	Provider        *string `json:"provider,omitempty"`
	Status          string  `json:"status"`
	FailureReason   *string `json:"failure_reason,omitempty"`
	AmountCFA       int     `json:"amount_cfa"`
	Currency        string  `json:"currency"`
	Description     *string `json:"description,omitempty"`
	RedirectURL     *string `json:"redirect_url,omitempty"`
	CallbackURL     *string `json:"callback_url,omitempty"`
	// Suivi du relais ABMCY Core -> app (voir migration 002).
	RelayStatus        string     `json:"relay_status"`
	RelayAttempts      int        `json:"relay_attempts"`
	RelayLastError     *string    `json:"relay_last_error,omitempty"`
	RelayLastAttemptAt *time.Time `json:"relay_last_attempt_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// PaymentWithApp — un Payment enrichi du nom de son app, pour le listing
// admin (évite un N+1 côté dashboard).
type PaymentWithApp struct {
	*Payment
	AppName string `json:"app_name"`
}

// CreatePaymentInput — POST /v1/pay (une app initie un paiement).
type CreatePaymentInput struct {
	AppRef      string `json:"app_ref"`
	AmountCFA   int    `json:"amount_cfa"`
	Country     string `json:"country"`
	Description string `json:"description,omitempty"`
	CallbackURL string `json:"callback_url,omitempty"`
}
