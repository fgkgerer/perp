package market

import (
	"math/big"
	"testing"
)

func TestLimitPriceWithSlippage_Buy(t *testing.T) {
	ref := big.NewInt(1_000_000)
	limit := LimitPriceWithSlippage(ref, true, 50)
	want := big.NewInt(1_005_000)
	if limit.Cmp(want) != 0 {
		t.Fatalf("got %s want %s", limit, want)
	}
}

func TestLimitPriceWithSlippage_Sell(t *testing.T) {
	ref := big.NewInt(1_000_000)
	limit := LimitPriceWithSlippage(ref, false, 100)
	want := big.NewInt(990_000)
	if limit.Cmp(want) != 0 {
		t.Fatalf("got %s want %s", limit, want)
	}
}

func TestParsePaperHuman(t *testing.T) {
	p, err := ParsePaperHuman("0.01")
	if err != nil {
		t.Fatal(err)
	}
	if p.String() != "10000000000000000" {
		t.Fatalf("got %s", p)
	}
}
