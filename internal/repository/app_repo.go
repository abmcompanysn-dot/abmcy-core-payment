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

const appColumns = `id, name, api_key_hash, hmac_secret_hash, default_callback_url, is_active, created_at, updated_at`

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

func (r *AppRepo) List(ctx context.Context) ([]*model.App, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+appColumns+` FROM apps ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*model.App{}
	for rows.Next() {
		a := &model.App{}
		if err := rows.Scan(&a.ID, &a.Name, &a.APIKeyHash, &a.HMACSecretHash, &a.DefaultCallbackURL, &a.IsActive, &a.CreatedAt, &a.UpdatedAt); err != nil {
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

func scanApp(row pgx.Row) (*model.App, error) {
	a := &model.App{}
	err := row.Scan(&a.ID, &a.Name, &a.APIKeyHash, &a.HMACSecretHash, &a.DefaultCallbackURL, &a.IsActive, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrAppNotFound
		}
		return nil, err
	}
	return a, nil
}

// --- Payments -------------------------------------------------------------

const paymentColumns = `id, app_id, app_ref, diarra_client_ref, type, provider, status, failure_reason,
	amount_cfa, currency, description, redirect_url, callback_url, created_at, updated_at`

func scanPayment(row pgx.Row) (*model.Payment, error) {
	p := &model.Payment{}
	err := row.Scan(&p.ID, &p.AppID, &p.AppRef, &p.DiarraClientRef, &p.Type, &p.Provider, &p.Status, &p.FailureReason,
		&p.AmountCFA, &p.Currency, &p.Description, &p.RedirectURL, &p.CallbackURL, &p.CreatedAt, &p.UpdatedAt)
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
	Currency        string
	Description     *string
	CallbackURL     *string
}

func (r *AppRepo) CreatePayment(ctx context.Context, p CreatePaymentParams) (*model.Payment, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO payments (app_id, app_ref, diarra_client_ref, amount_cfa, currency, description, callback_url)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING `+paymentColumns,
		p.AppID, p.AppRef, p.DiarraClientRef, p.AmountCFA, p.Currency, p.Description, p.CallbackURL)
	return scanPayment(row)
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
