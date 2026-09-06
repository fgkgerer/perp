package engine

import (
	"math/big"
	"testing"
	"time"
)

func TestSimulateTakerFill_BuyWalksAsks(t *testing.T) {
	e := newTestEngine()
	now := time.Now()
	mustAddOrder(t, e, testOrder("ask1", "0x00000000000000000000000000000000000000a1", "-20", "2000", now))
	mustAddOrder(t, e, testOrder("ask2", "0x00000000000000000000000000000000000000a2", "-20", "2200", now))

	e.mu.RLock()
	book := e.orderBooks[testPerp]
	e.mu.RUnlock()
	if book == nil {
		t.Fatal("missing book")
	}

	size := big.NewInt(30)
	limit := calculatePrice(testOrder("ask2", "0x00000000000000000000000000000000000000a2", "-20", "2200", now))

	sim := book.SimulateTakerFill(true, size, limit, "")
	if sim.FilledPaper.Cmp(size) != 0 {
		t.Fatalf("filled=%s want=%s", sim.FilledPaper, size)
	}
	if len(sim.Levels) != 2 {
		t.Fatalf("levels=%d want 2", len(sim.Levels))
	}
	if sim.WorstPriceRaw == nil || sim.AvgPriceRaw == nil {
		t.Fatal("expected avg/worst price")
	}
}

func TestSimulateTakerFill_LimitBlocksHigherAsk(t *testing.T) {
	e := newTestEngine()
	now := time.Now()
	mustAddOrder(t, e, testOrder("ask1", "0x00000000000000000000000000000000000000a1", "-20", "2000", now))
	mustAddOrder(t, e, testOrder("ask2", "0x00000000000000000000000000000000000000a2", "-20", "2200", now))

	e.mu.RLock()
	book := e.orderBooks[testPerp]
	e.mu.RUnlock()

	size := big.NewInt(30)
	limit := calculatePrice(testOrder("ask1", "0x00000000000000000000000000000000000000a1", "-20", "2000", now))

	sim := book.SimulateTakerFill(true, size, limit, "")
	if sim.FilledPaper.Cmp(big.NewInt(20)) != 0 {
		t.Fatalf("filled=%s want 20", sim.FilledPaper)
	}
	if sim.UnfilledPaper.Cmp(big.NewInt(10)) != 0 {
		t.Fatalf("unfilled=%s want 10", sim.UnfilledPaper)
	}
}
