package fees

import "testing"

func TestCompute(t *testing.T) {
	cases := []struct {
		amount, rate     int
		wantFee, wantNet int
	}{
		{0, 600, 0, 0},
		{100, 600, 50, 50},            // <1$ -> min 1$ de commission
		{600, 600, 50, 550},           // pile 1$
		{601, 600, 100, 501},          // 2$ entamés
		{6000, 600, 500, 5500},        // 10$
		{200_000, 600, 16700, 183300}, // ceil(200000/600)=334 -> 334*50
		{40, 600, 40, 0},              // commission plafonnée au montant
	}
	for _, c := range cases {
		got := Compute(c.amount, c.rate)
		if got.FeeCFA != c.wantFee || got.NetCFA != c.wantNet {
			t.Errorf("Compute(%d,%d) = fee %d net %d ; want fee %d net %d",
				c.amount, c.rate, got.FeeCFA, got.NetCFA, c.wantFee, c.wantNet)
		}
	}
}

func TestComputeRateFallback(t *testing.T) {
	if got := Compute(600, 0); got.USDRateUsed != DefaultUSDRate {
		t.Errorf("taux 0 -> %d, want %d", got.USDRateUsed, DefaultUSDRate)
	}
}
