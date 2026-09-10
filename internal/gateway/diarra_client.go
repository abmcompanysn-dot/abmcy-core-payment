// Package gateway — client HTTP vers la passerelle de paiement DIARRA
// (POST /api/gateway/v1/*). ABMCY Core n'intègre PawaPay/KPay/PayPal nulle
// part : tout passe par DIARRA, qui garde la seule intégration agrégateur
// (voir backend/internal/payment/provider.go côté DIARRA).
//
// Authentification en deux temps sur chaque requête, symétrique à ce que
// DIARRA vérifie (voir backend/internal/middleware/gateway.go) :
//  1. X-Gateway-Key : la clé API du client (identifie ABMCY Core).
//  2. X-Diarra-Signature + X-Diarra-Timestamp : HMAC-SHA256(hash(secret),
//     timestamp + "." + corps), preuve que CE client précis a signé CE
//     corps précis, avec fenêtre de validité anti-rejeu de 5 min.
package gateway

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type Config struct {
	BaseURL string // ex: "https://core.diarra.app" une fois déployé, "http://localhost:8080" en local
	APIKey  string // X-Gateway-Key, remis en clair UNE SEULE FOIS à la création du client côté DIARRA
	// HMACSecretHash : le HASH SHA-256 (hex) du secret HMAC remis à la
	// création du client — DIARRA vérifie contre ce même hash utilisé comme
	// clé HMAC (jamais le secret brut, voir le commentaire sur
	// verifyGatewaySignature côté DIARRA). Calculer une fois au démarrage :
	// hex.EncodeToString(sha256.Sum256([]byte(secretBrut))[:]).
	HMACSecretHash string
}

type Client struct {
	cfg    Config
	client *http.Client
}

func New(cfg Config) *Client {
	return &Client{cfg: cfg, client: &http.Client{Timeout: 30 * time.Second}}
}

// --- Dépôts (paiement entrant) ------------------------------------------

type CreateDepositInput struct {
	ClientRef   string `json:"client_ref"`
	AmountCFA   int    `json:"amount_cfa"`
	Country     string `json:"country"`
	Description string `json:"description,omitempty"`
	CallbackURL string `json:"callback_url,omitempty"`
	// ReturnURL : où DIARRA renvoie le navigateur après la page de paiement.
	// ABMCY Core y met sa propre page de suivi /pay/{ref} (qui redirige
	// ensuite vers la return_url du marchand).
	ReturnURL string `json:"return_url,omitempty"`
}

type Transaction struct {
	ID            string `json:"id,omitempty"` // non exposé par DIARRA (interne) — absent en pratique
	ClientRef     string `json:"client_ref"`
	Type          string `json:"type"`
	Provider      string `json:"provider"`
	ProviderRef   string `json:"provider_ref,omitempty"`
	Status        string `json:"status"` // pending|processing|completed|failed|cancelled
	FailureReason string `json:"failure_reason,omitempty"`
	AmountCFA     int    `json:"amount_cfa"`
	Currency      string `json:"currency"`
	RedirectURL   string `json:"redirect_url,omitempty"`
	CreatedAt     string `json:"created_at"`
}

type txResponse struct {
	Transaction Transaction `json:"transaction"`
}

func (c *Client) CreateDeposit(input CreateDepositInput) (*Transaction, error) {
	var out txResponse
	if err := c.doSigned(http.MethodPost, "/api/gateway/v1/deposits", input, &out); err != nil {
		return nil, err
	}
	return &out.Transaction, nil
}

func (c *Client) GetDeposit(clientRef string) (*Transaction, error) {
	var out txResponse
	if err := c.doSigned(http.MethodGet, "/api/gateway/v1/deposits/"+clientRef, nil, &out); err != nil {
		return nil, err
	}
	return &out.Transaction, nil
}

// --- Versements sortants ---------------------------------------------------

type CreatePayoutInput struct {
	ClientRef         string `json:"client_ref"`
	AmountCFA         int    `json:"amount_cfa"`
	RecipientPhone    string `json:"recipient_phone"`
	RecipientOperator string `json:"recipient_operator"`
	Country           string `json:"country"`
	CallbackURL       string `json:"callback_url,omitempty"`
}

func (c *Client) CreatePayout(input CreatePayoutInput) (*Transaction, error) {
	var out txResponse
	if err := c.doSigned(http.MethodPost, "/api/gateway/v1/payouts", input, &out); err != nil {
		return nil, err
	}
	return &out.Transaction, nil
}

func (c *Client) GetPayout(clientRef string) (*Transaction, error) {
	var out txResponse
	if err := c.doSigned(http.MethodGet, "/api/gateway/v1/payouts/"+clientRef, nil, &out); err != nil {
		return nil, err
	}
	return &out.Transaction, nil
}

// --- Remboursements ----------------------------------------------------

type CreateRefundInput struct {
	ClientRef        string `json:"client_ref"`
	DepositClientRef string `json:"deposit_client_ref"`
	CallbackURL      string `json:"callback_url,omitempty"`
}

func (c *Client) CreateRefund(input CreateRefundInput) (*Transaction, error) {
	var out txResponse
	if err := c.doSigned(http.MethodPost, "/api/gateway/v1/refunds", input, &out); err != nil {
		return nil, err
	}
	return &out.Transaction, nil
}

// --- Signature + appel HTTP ---------------------------------------------

func (c *Client) doSigned(method, path string, body interface{}, out interface{}) error {
	var raw []byte
	if body != nil {
		var err error
		raw, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}

	ts := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(c.cfg.HMACSecretHash))
	mac.Write([]byte(ts + "." + string(raw)))
	signature := hex.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequest(method, c.cfg.BaseURL+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gateway-Key", c.cfg.APIKey)
	req.Header.Set("X-Diarra-Timestamp", ts)
	req.Header.Set("X-Diarra-Signature", signature)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("diarra gateway: status %d: %s", resp.StatusCode, string(respBody))
	}
	if out != nil {
		return json.Unmarshal(respBody, out)
	}
	return nil
}
