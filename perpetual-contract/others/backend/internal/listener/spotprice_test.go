package listener

import (
	"math/big"
	"testing"
)

func TestWeightedSpotUSD_433(t *testing.T) {
	cb := big.NewFloat(100)
	okx := big.NewFloat(90)
	bn := big.NewFloat(80)
	out, ok := WeightedSpotUSD(cb, okx, bn, SpotWeights{4, 3, 3})
	if !ok {
		t.Fatal("expected ok")
	}
	// (100*4 + 90*3 + 80*3) / 10 = 910/10 = 91
	want := big.NewFloat(91)
	if out.Cmp(want) != 0 {
		t.Fatalf("got %s want 91", out.Text('f', 2))
	}
}

func TestWeightedSpotUSD_renormalize(t *testing.T) {
	cb := big.NewFloat(100)
	out, ok := WeightedSpotUSD(cb, nil, nil, SpotWeights{4, 3, 3})
	if !ok || out.Cmp(cb) != 0 {
		t.Fatalf("single venue should return its price, got %v ok=%v", out, ok)
	}
}

func TestFormatUsdDisplay2(t *testing.T) {
	if FormatUsdDisplay2(big.NewFloat(30000.126)) != "30000.13" {
		t.Fatal("rounding")
	}
	if FormatIndexDisplayFrom1e6("30000126000") != "30000.13" {
		t.Fatal("from 1e6")
	}
}
