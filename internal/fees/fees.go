// Package fees — calcul de la commission ABMCY Core sur une transaction.
//
// Règle actuelle : 50 FCFA par dollar de transaction (le "dollar" est une
// unité de compte interne, taux F/USD configurable via ABMCY_USD_RATE,
// défaut 600). La commission est figée au moment de la création du paiement
// (le taux peut bouger ensuite) et retenue sur le montant reversé au
// marchand : net = montant - commission.
//
// Les frais des agrégateurs et du mobile money ne sont PAS encore inclus
// ici (ils s'ajouteront quand la grille exacte sera connue).
package fees

const (
	// FeePerUSD : marge ABMCY Core, en FCFA, par dollar de transaction.
	FeePerUSD = 50
	// DefaultUSDRate : F/USD par défaut si ABMCY_USD_RATE absent/invalide.
	DefaultUSDRate = 600
)

// Result — décomposition d'une transaction après commission.
type Result struct {
	AmountCFA   int
	FeeCFA      int
	NetCFA      int
	USDRateUsed int
}

// Compute calcule la commission pour un montant en FCFA à un taux F/USD
// donné. La commission est arrondie au FCFA supérieur par tranche de dollar
// entamée (ceil), avec un minimum de FeePerUSD (toute transaction non nulle
// coûte au moins un "dollar" de commission). Ne descend jamais le net en
// dessous de zéro.
func Compute(amountCFA, usdRate int) Result {
	if usdRate <= 0 {
		usdRate = DefaultUSDRate
	}
	if amountCFA <= 0 {
		return Result{AmountCFA: amountCFA, FeeCFA: 0, NetCFA: amountCFA, USDRateUsed: usdRate}
	}
	// nombre de dollars entamés = ceil(amount / rate)
	dollars := (amountCFA + usdRate - 1) / usdRate
	if dollars < 1 {
		dollars = 1
	}
	fee := dollars * FeePerUSD
	if fee > amountCFA {
		fee = amountCFA
	}
	return Result{
		AmountCFA:   amountCFA,
		FeeCFA:      fee,
		NetCFA:      amountCFA - fee,
		USDRateUsed: usdRate,
	}
}
