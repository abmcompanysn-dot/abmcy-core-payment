package model

import "time"

const (
	SignupPending  = "pending"
	SignupApproved = "approved"
	SignupRejected = "rejected"
)

// SignupRequest — une demande d'ouverture de compte depuis la landing page
// publique, en attente de validation manuelle. Voir migration 005.
type SignupRequest struct {
	ID             string     `json:"id"`
	BusinessName   string     `json:"business_name"`
	ContactName    string     `json:"contact_name"`
	Email          string     `json:"email"`
	Phone          *string    `json:"phone,omitempty"`
	Website        *string    `json:"website,omitempty"`
	Country        *string    `json:"country,omitempty"`
	Description    *string    `json:"description,omitempty"`
	ExpectedVolume *string    `json:"expected_volume,omitempty"`
	Status         string     `json:"status"`
	ReviewNote     *string    `json:"review_note,omitempty"`
	AppID          *string    `json:"app_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	ReviewedAt     *time.Time `json:"reviewed_at,omitempty"`
}

// PublicSignupInput — corps de POST /public/signup (formulaire landing).
type PublicSignupInput struct {
	BusinessName   string `json:"business_name"`
	ContactName    string `json:"contact_name"`
	Email          string `json:"email"`
	Phone          string `json:"phone,omitempty"`
	Website        string `json:"website,omitempty"`
	Country        string `json:"country,omitempty"`
	Description    string `json:"description,omitempty"`
	ExpectedVolume string `json:"expected_volume,omitempty"`
}
