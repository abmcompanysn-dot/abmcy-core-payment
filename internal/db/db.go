// Package db — connexion Postgres d'ABMCY Core (base abmcy_core, logiquement
// séparée de celle de DIARRA mais sur le même serveur diarra-vps). pgxpool,
// même choix que DIARRA backend.
package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect ouvre un pool Postgres à partir de DATABASE_URL et vérifie la
// connexion (Ping) avant de rendre la main — échec rapide au démarrage
// plutôt qu'à la première requête.
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 10
	cfg.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
