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
	KYCLevel           string    `json:"kyc_level"` // "none" | "verified"
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

const (
	KYCNone     = "none"
	KYCVerified = "verified"

	// Plafonds par transaction, en FCFA, selon le niveau KYC. Au-delà,
	// /v1/pay refuse (limit_exceeded).
	MaxAmountNoKYC    = 200_000
	MaxAmountVerified = 1_000_000
)

// MaxAmountFor renvoie le plafond par transaction pour un niveau KYC donné.
func MaxAmountFor(kycLevel string) int {
	if kycLevel == KYCVerified {
		return MaxAmountVerified
	}
	return MaxAmountNoKYC
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
	// FeeCFA : commission ABMCY Core figée à la création. NetCFA = AmountCFA
	// - FeeCFA (reversé au marchand). Voir migration 004 / fees.Compute.
	FeeCFA      int     `json:"fee_cfa"`
	NetCFA      *int    `json:"net_cfa,omitempty"`
	USDRateUsed *int    `json:"usd_rate_used,omitempty"`
	Currency    string  `json:"currency"`
	Description *string `json:"description,omitempty"`
	RedirectURL *string `json:"redirect_url,omitempty"`
	CallbackURL *string `json:"callback_url,omitempty"`
	// ReturnURL : où renvoyer le NAVIGATEUR de l'utilisateur final une fois
	// le paiement terminé (page hébergée / widget). Distincte de CallbackURL
	// (notification serveur-à-serveur). Voir migration 003.
	ReturnURL *string `json:"return_url,omitempty"`
	// RefundOfPaymentID : pour un Payment de type "refund", l'ID du paiement
	// (dépôt) remboursé. Voir migration 006.
	RefundOfPaymentID *string `json:"refund_of_payment_id,omitempty"`
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
	ReturnURL   string `json:"return_url,omitempty"`
}

// CreateRefundInput — POST /v1/refunds (une app rembourse un paiement).
// DepositAppRef = l'app_ref du dépôt d'origine (déjà "completed").
type CreateRefundInput struct {
	DepositAppRef string `json:"deposit_app_ref"`
	RefundAppRef  string `json:"refund_app_ref,omitempty"` // optionnel, défaut "refund-<deposit_app_ref>"
	CallbackURL   string `json:"callback_url,omitempty"`
}
