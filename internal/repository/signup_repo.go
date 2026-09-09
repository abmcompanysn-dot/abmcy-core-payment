package repository

import (
	"context"
	"errors"

	"github.com/abmcy/core/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// isUniqueViolation — code SQLSTATE 23505 (violation d'unicité), pour
// distinguer "demande déjà en attente" d'une vraie erreur.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

var (
	ErrSignupNotFound      = errors.New("signup request not found")
	ErrSignupPendingExists = errors.New("une demande est déjà en attente pour cet email")
)

type SignupRepo struct {
	pool *pgxpool.Pool
}

func NewSignupRepo(pool *pgxpool.Pool) *SignupRepo {
	return &SignupRepo{pool: pool}
}

const signupColumns = `id, business_name, contact_name, email, phone, website, country,
	description, expected_volume, status, review_note, app_id, created_at, reviewed_at`

func scanSignup(row pgx.Row) (*model.SignupRequest, error) {
	s := &model.SignupRequest{}
	err := row.Scan(&s.ID, &s.BusinessName, &s.ContactName, &s.Email, &s.Phone, &s.Website, &s.Country,
		&s.Description, &s.ExpectedVolume, &s.Status, &s.ReviewNote, &s.AppID, &s.CreatedAt, &s.ReviewedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSignupNotFound
		}
		return nil, err
	}
	return s, nil
}

func strOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Create enregistre une nouvelle demande. Renvoie ErrSignupPendingExists si
// une demande 'pending' existe déjà pour cet email (index unique partiel).
func (r *SignupRepo) Create(ctx context.Context, in model.PublicSignupInput) (*model.SignupRequest, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO signup_requests
		   (business_name, contact_name, email, phone, website, country, description, expected_volume)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 RETURNING `+signupColumns,
		in.BusinessName, in.ContactName, in.Email,
		strOrNil(in.Phone), strOrNil(in.Website), strOrNil(in.Country),
		strOrNil(in.Description), strOrNil(in.ExpectedVolume))
	s, err := scanSignup(row)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrSignupPendingExists
		}
		return nil, err
	}
	return s, nil
}

// List renvoie les demandes, filtrable par statut ("" = toutes), récentes
// d'abord.
func (r *SignupRepo) List(ctx context.Context, status string) ([]*model.SignupRequest, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+signupColumns+` FROM signup_requests
		 WHERE ($1 = '' OR status = $1)
		 ORDER BY created_at DESC LIMIT 500`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*model.SignupRequest{}
	for rows.Next() {
		s := &model.SignupRequest{}
		if err := rows.Scan(&s.ID, &s.BusinessName, &s.ContactName, &s.Email, &s.Phone, &s.Website, &s.Country,
			&s.Description, &s.ExpectedVolume, &s.Status, &s.ReviewNote, &s.AppID, &s.CreatedAt, &s.ReviewedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *SignupRepo) FindByID(ctx context.Context, id string) (*model.SignupRequest, error) {
	return scanSignup(r.pool.QueryRow(ctx, `SELECT `+signupColumns+` FROM signup_requests WHERE id = $1`, id))
}

// MarkApproved lie la demande à l'app créée et passe le statut à 'approved'.
func (r *SignupRepo) MarkApproved(ctx context.Context, id, appID, note string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE signup_requests
		   SET status = 'approved', app_id = $2, review_note = NULLIF($3,''), reviewed_at = now()
		 WHERE id = $1`, id, appID, note)
	return err
}

func (r *SignupRepo) MarkRejected(ctx context.Context, id, note string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE signup_requests
		   SET status = 'rejected', review_note = NULLIF($2,''), reviewed_at = now()
		 WHERE id = $1`, id, note)
	return err
}
