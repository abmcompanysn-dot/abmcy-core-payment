package repository

import (
	"context"
	"errors"

	"github.com/abmcy/core/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAppNotFound     = errors.New("app not found")
	ErrPaymentNotFound = errors.New("payment not found")
)

type AppRepo struct {
	pool *pgxpool.Pool
}

func NewAppRepo(pool *pgxpool.Pool) *AppRepo {
	return &AppRepo{pool: pool}
}

const appColumns = `id, name, api_key_hash, hmac_secret_hash, default_callback_url, is_active, kyc_level, created_at, updated_at`

func (r *AppRepo) Create(ctx context.Context, name, apiKeyHash, hmacSecretHash string, defaultCallbackURL *string) (*model.App, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO apps (name, api_key_hash, hmac_secret_hash, default_callback_url)
		 VALUES ($1, $2, $3, $4) RETURNING `+appColumns,
		name, apiKeyHash, hmacSecretHash, defaultCallbackURL)
	return scanApp(row)
}

func (r *AppRepo) FindByKeyHash(ctx context.Context, keyHash string) (*model.App, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+appColumns+` FROM apps WHERE api_key_hash = $1 AND is_active = TRUE`, keyHash)
	return scanApp(row)
}

// FindByID — usage interne (signature d'un relais sortant, dashboard admin).
// Ne filtre PAS sur is_active : un relais sur un paiement d'une app depuis
// désactivée reste légitime.
func (r *AppRepo) FindByID(ctx context.Context, id string) (*model.App, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+appColumns+` FROM apps WHERE id = $1`, id)
	return scanApp(row)
}

func (r *AppRepo) List(ctx context.Context) ([]*model.App, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+appColumns+` FROM apps ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*model.App{}
	for rows.Next() {
		a, err := scanAppRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *AppRepo) SetActive(ctx context.Context, id string, active bool) error {
	_, err := r.pool.Exec(ctx, `UPDATE apps SET is_active = $2, updated_at = now() WHERE id = $1`, id, active)
	return err
}

// SetKYCLevel — "none" | "verified" (voir model.KYC*). Validé à la main
// depuis le back-office admin.
func (r *AppRepo) SetKYCLevel(ctx context.Context, id, level string) error {
	_, err := r.pool.Exec(ctx, `UPDATE apps SET kyc_level = $2, updated_at = now() WHERE id = $1`, id, level)
	return err
}

func scanApp(row pgx.Row) (*model.App, error) {
	a := &model.App{}
	err := row.Scan(&a.ID, &a.Name, &a.APIKeyHash, &a.HMACSecretHash, &a.DefaultCallbackURL, &a.IsActive, &a.KYCLevel, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrAppNotFound
		}
		return nil, err
	}
	return a, nil
}

func scanAppRows(rows pgx.Rows) (*model.App, error) {
	a := &model.App{}
	err := rows.Scan(&a.ID, &a.Name, &a.APIKeyHash, &a.HMACSecretHash, &a.DefaultCallbackURL, &a.IsActive, &a.KYCLevel, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

// --- Payments -------------------------------------------------------------

const paymentColumns = `id, app_id, app_ref, diarra_client_ref, type, provider, status, failure_reason,
	amount_cfa, fee_cfa, net_cfa, usd_rate_used, currency, description, redirect_url, callback_url, return_url,
	relay_status, relay_attempts, relay_last_error, relay_last_attempt_at,
	created_at, updated_at`

// Mêmes colonnes que paymentColumns, préfixées "p." pour les jointures.
const paymentColumnsP = `p.id, p.app_id, p.app_ref, p.diarra_client_ref, p.type, p.provider, p.status, p.failure_reason,
	p.amount_cfa, p.fee_cfa, p.net_cfa, p.usd_rate_used, p.currency, p.description, p.redirect_url, p.callback_url, p.return_url,
	p.relay_status, p.relay_attempts, p.relay_last_error, p.relay_last_attempt_at,
	p.created_at, p.updated_at`

// paymentScanTargets — cibles de Scan dans l'ordre de paymentColumns /
// paymentColumnsP. Un seul endroit à maintenir quand le schéma bouge.
func paymentScanTargets(p *model.Payment) []any {
	return []any{
		&p.ID, &p.AppID, &p.AppRef, &p.DiarraClientRef, &p.Type, &p.Provider, &p.Status, &p.FailureReason,
		&p.AmountCFA, &p.FeeCFA, &p.NetCFA, &p.USDRateUsed, &p.Currency, &p.Description, &p.RedirectURL, &p.CallbackURL, &p.ReturnURL,
		&p.RelayStatus, &p.RelayAttempts, &p.RelayLastError, &p.RelayLastAttemptAt,
		&p.CreatedAt, &p.UpdatedAt,
	}
}

func scanPaymentRow(row pgx.Row) (*model.Payment, error) {
	p := &model.Payment{}
	err := row.Scan(paymentScanTargets(p)...)
	return p, err
}

func scanPayment(row pgx.Row) (*model.Payment, error) {
	p, err := scanPaymentRow(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrPaymentNotFound
		}
		return nil, err
	}
	return p, nil
}

type CreatePaymentParams struct {
	AppID           string
	AppRef          string
	DiarraClientRef string
	AmountCFA       int
	FeeCFA          int
	NetCFA          int
	USDRateUsed     int
	Currency        string
	Description     *string
	CallbackURL     *string
	ReturnURL       *string
}

func (r *AppRepo) CreatePayment(ctx context.Context, p CreatePaymentParams) (*model.Payment, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO payments (app_id, app_ref, diarra_client_ref, amount_cfa, fee_cfa, net_cfa, usd_rate_used,
		                       currency, description, callback_url, return_url)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING `+paymentColumns,
		p.AppID, p.AppRef, p.DiarraClientRef, p.AmountCFA, p.FeeCFA, p.NetCFA, p.USDRateUsed,
		p.Currency, p.Description, p.CallbackURL, p.ReturnURL)
	return scanPayment(row)
}

// FindPaymentByDiarraRefPublic — lecture SANS auth pour la page de paiement
// hébergée (/pay/{ref}). Renvoie aussi le nom de l'app pour l'affichage.
func (r *AppRepo) FindPaymentByDiarraRefPublic(ctx context.Context, diarraClientRef string) (*model.PaymentWithApp, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+paymentColumnsP+`, a.name
		 FROM payments p JOIN apps a ON a.id = p.app_id
		 WHERE p.diarra_client_ref = $1`, diarraClientRef)
	pw := &model.PaymentWithApp{Payment: &model.Payment{}}
	err := row.Scan(append(paymentScanTargets(pw.Payment), &pw.AppName)...)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrPaymentNotFound
		}
		return nil, err
	}
	return pw, nil
}

func (r *AppRepo) FindPaymentByAppRef(ctx context.Context, appID, appRef string) (*model.Payment, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+paymentColumns+` FROM payments WHERE app_id = $1 AND app_ref = $2`, appID, appRef)
	return scanPayment(row)
}

func (r *AppRepo) FindPaymentByDiarraRef(ctx context.Context, diarraClientRef string) (*model.Payment, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+paymentColumns+` FROM payments WHERE diarra_client_ref = $1`, diarraClientRef)
	return scanPayment(row)
}

func (r *AppRepo) UpdatePaymentRedirect(ctx context.Context, id, redirectURL string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE payments SET redirect_url = $2, updated_at = now() WHERE id = $1`, id, redirectURL)
	return err
}

func (r *AppRepo) UpdatePaymentStatus(ctx context.Context, id, status string, provider, failureReason *string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE payments SET status = $2, provider = COALESCE($3, provider), failure_reason = $4, updated_at = now() WHERE id = $1`,
		id, status, provider, failureReason)
	return err
}

// FindPaymentByID — lecture admin d'un paiement précis (dashboard).
func (r *AppRepo) FindPaymentByID(ctx context.Context, id string) (*model.Payment, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+paymentColumns+` FROM payments WHERE id = $1`, id)
	return scanPayment(row)
}

// ListPaymentsFilter — filtres du listing admin. Champs vides = pas de filtre.
type ListPaymentsFilter struct {
	AppID  string
	Status string
	Limit  int
	Offset int
}

// ListPayments — listing admin paginé, le plus récent d'abord. Renvoie aussi
// le nom de l'app (jointure) pour éviter un N+1 côté dashboard.
func (r *AppRepo) ListPayments(ctx context.Context, f ListPaymentsFilter) ([]*model.PaymentWithApp, error) {
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx,
		`SELECT `+paymentColumnsP+`, a.name
		 FROM payments p JOIN apps a ON a.id = p.app_id
		 WHERE ($1 = '' OR p.app_id = $1::uuid)
		   AND ($2 = '' OR p.status = $2)
		 ORDER BY p.created_at DESC
		 LIMIT $3 OFFSET $4`,
		f.AppID, f.Status, limit, f.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*model.PaymentWithApp{}
	for rows.Next() {
		pw := &model.PaymentWithApp{Payment: &model.Payment{}}
		if err := rows.Scan(append(paymentScanTargets(pw.Payment), &pw.AppName)...); err != nil {
			return nil, err
		}
		out = append(out, pw)
	}
	return out, rows.Err()
}

// SetRelayResult enregistre l'issue d'une tentative de relais vers l'app
// (voir PaymentHandler.relayToApp). errMsg == nil => succès.
func (r *AppRepo) SetRelayResult(ctx context.Context, id, status string, errMsg *string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE payments
		   SET relay_status = $2,
		       relay_attempts = relay_attempts + 1,
		       relay_last_error = $3,
		       relay_last_attempt_at = now(),
		       updated_at = now()
		 WHERE id = $1`,
		id, status, errMsg)
	return err
}
