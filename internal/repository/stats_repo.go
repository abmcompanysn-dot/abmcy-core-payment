package repository

import (
	"context"
	"time"
)

// --- Tableau de bord chiffré (méthode sur AppRepo) -----------------------

type Totals struct {
	Count       int     `json:"count"`
	AmountCFA   int64   `json:"amount_cfa"`
	FeeCFA      int64   `json:"fee_cfa"`
	NetCFA      int64   `json:"net_cfa"`
	Completed   int     `json:"completed"`
	Failed      int     `json:"failed"`
	SuccessRate float64 `json:"success_rate"` // 0..1
}

type ByType struct {
	Type      string `json:"type"`
	Count     int    `json:"count"`
	AmountCFA int64  `json:"amount_cfa"`
	FeeCFA    int64  `json:"fee_cfa"`
}

type ByApp struct {
	AppID       string  `json:"app_id"`
	AppName     string  `json:"app_name"`
	Count       int     `json:"count"`
	AmountCFA   int64   `json:"amount_cfa"`
	FeeCFA      int64   `json:"fee_cfa"`
	SuccessRate float64 `json:"success_rate"`
}

type ByOperator struct {
	Operator  string `json:"operator"`
	Count     int    `json:"count"`
	AmountCFA int64  `json:"amount_cfa"`
}

type Dashboard struct {
	SinceDays  int          `json:"since_days"` // 0 = tout
	Totals     Totals       `json:"totals"`
	ByType     []ByType     `json:"by_type"`
	ByApp      []ByApp      `json:"by_app"`
	ByOperator []ByOperator `json:"by_operator"`
}

// Dashboard agrège tout sur `sinceDays` jours (0 = depuis toujours). Ne
// compte que les transactions dont le statut est terminal ou en cours —
// exclut celles jamais parties (aucun provider) pour ne pas gonfler les
// volumes. On considère "abouties" celles au statut 'completed'.
func (r *AppRepo) Dashboard(ctx context.Context, sinceDays int) (*Dashboard, error) {
	var since time.Time
	// filterP : condition sur la table payments seule (alias implicite).
	// filterPJoin : idem mais qualifiée p. pour les requêtes avec JOIN apps
	// (apps a aussi une colonne created_at -> ambiguïté sinon).
	filterP, filterPJoin := "TRUE", "TRUE"
	args := []any{}
	if sinceDays > 0 {
		since = time.Now().AddDate(0, 0, -sinceDays)
		filterP = "created_at >= $1"
		filterPJoin = "p.created_at >= $1"
		args = append(args, since)
	}

	d := &Dashboard{SinceDays: sinceDays}

	// Totaux
	row := r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COALESCE(SUM(amount_cfa),0),
			COALESCE(SUM(fee_cfa),0),
			COALESCE(SUM(COALESCE(net_cfa, amount_cfa - fee_cfa)),0),
			COUNT(*) FILTER (WHERE status='completed'),
			COUNT(*) FILTER (WHERE status='failed')
		FROM payments WHERE `+filterP, args...)
	if err := row.Scan(&d.Totals.Count, &d.Totals.AmountCFA, &d.Totals.FeeCFA, &d.Totals.NetCFA,
		&d.Totals.Completed, &d.Totals.Failed); err != nil {
		return nil, err
	}
	if d.Totals.Completed+d.Totals.Failed > 0 {
		d.Totals.SuccessRate = float64(d.Totals.Completed) / float64(d.Totals.Completed+d.Totals.Failed)
	}

	// Par type
	rows, err := r.pool.Query(ctx, `
		SELECT type, COUNT(*), COALESCE(SUM(amount_cfa),0), COALESCE(SUM(fee_cfa),0)
		FROM payments WHERE `+filterP+`
		GROUP BY type ORDER BY type`, args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var t ByType
		if err := rows.Scan(&t.Type, &t.Count, &t.AmountCFA, &t.FeeCFA); err != nil {
			rows.Close()
			return nil, err
		}
		d.ByType = append(d.ByType, t)
	}
	rows.Close()

	// Par app
	rows, err = r.pool.Query(ctx, `
		SELECT p.app_id, a.name, COUNT(*), COALESCE(SUM(p.amount_cfa),0), COALESCE(SUM(p.fee_cfa),0),
			COUNT(*) FILTER (WHERE p.status='completed'),
			COUNT(*) FILTER (WHERE p.status IN ('completed','failed'))
		FROM payments p JOIN apps a ON a.id = p.app_id
		WHERE `+filterPJoin+`
		GROUP BY p.app_id, a.name
		ORDER BY SUM(p.fee_cfa) DESC`, args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var b ByApp
		var comp, term int
		if err := rows.Scan(&b.AppID, &b.AppName, &b.Count, &b.AmountCFA, &b.FeeCFA, &comp, &term); err != nil {
			rows.Close()
			return nil, err
		}
		if term > 0 {
			b.SuccessRate = float64(comp) / float64(term)
		}
		d.ByApp = append(d.ByApp, b)
	}
	rows.Close()

	// Par opérateur (payouts uniquement — c'est là qu'on connaît l'opérateur ;
	// pour les dépôts, le provider technique n'est pas exposé).
	rows, err = r.pool.Query(ctx, `
		SELECT COALESCE(recipient_operator,'—'), COUNT(*), COALESCE(SUM(amount_cfa),0)
		FROM payments
		WHERE type='payout' AND recipient_operator IS NOT NULL AND `+filterP+`
		GROUP BY recipient_operator ORDER BY COUNT(*) DESC`, args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var o ByOperator
		if err := rows.Scan(&o.Operator, &o.Count, &o.AmountCFA); err != nil {
			rows.Close()
			return nil, err
		}
		d.ByOperator = append(d.ByOperator, o)
	}
	rows.Close()

	return d, nil
}
