// OrderBook 价档+FIFO 结构测试（orderbook.go）
//
// 验证与撮合无关的簿内行为：价档聚合、撤单索引、同价时间序。
package engine

import (
	"testing"
	"time"
)

// TestPriceLevelFIFOAndSnapshot
// 同价两单聚合到一档 totalRem；不同价分档排序；撤单后 totalRem 递减。
func TestPriceLevelFIFOAndSnapshot(t *testing.T) {
	book := newOrderBook(testPerp)
	now := time.Now()

	o1 := testOrder("a1", "0x00000000000000000000000000000000000000a1", "5", "-500", now)  // 买 @100
	o2 := testOrder("a2", "0x00000000000000000000000000000000000000a2", "3", "-300", now)  // 买 @100
	o3 := testOrder("a3", "0x00000000000000000000000000000000000000a3", "10", "-900", now.Add(time.Second)) // 买 @90

	book.insertResting(o1, absPaperString(o1.PaperAmount))
	book.insertResting(o2, absPaperString(o2.PaperAmount))
	book.insertResting(o3, absPaperString(o3.PaperAmount))

	bids := book.snapshotBids(10)
	if len(bids) != 2 {
		t.Fatalf("expected 2 bid levels, got %d: %+v", len(bids), bids)
	}
	// 最优档 @100：5+3=8；次档 @90：10
	if bids[0].Amount != "8" {
		t.Fatalf("expected aggregated amount 8 at best level, got %s", bids[0].Amount)
	}
	if bids[1].Amount != "10" {
		t.Fatalf("expected second level amount 10, got %s", bids[1].Amount)
	}

	if !book.remove("a1") {
		t.Fatal("expected remove a1 ok")
	}
	bids = book.snapshotBids(10)
	if bids[0].Amount != "3" {
		t.Fatalf("expected level amount 3 after remove, got %s", bids[0].Amount)
	}
}

// TestCancelIsO1WithIndex
// entries[orderId] 索引存在时，中间位置撤单不应影响其他订单。
func TestCancelIsO1WithIndex(t *testing.T) {
	book := newOrderBook(testPerp)
	now := time.Now()
	for i := 0; i < 50; i++ {
		o := testOrder("o-"+string(rune('a'+i)), "0x00000000000000000000000000000000000000b1", "1", "-100", now.Add(time.Duration(i)*time.Millisecond))
		book.insertResting(o, absPaperString(o.PaperAmount))
	}

	target := testOrder("target", "0x00000000000000000000000000000000000000c1", "1", "-100", now)
	book.insertResting(target, absPaperString(target.PaperAmount))
	if !book.remove("target") {
		t.Fatal("cancel target failed")
	}
	if book.hasOrder("target") {
		t.Fatal("target still in book")
	}
}

// TestMatchUsesMakerTimePriorityAtSamePrice
// 端到端：价档内按 CreateTime 排队，maker 为更早的卖单。
func TestMatchUsesMakerTimePriorityAtSamePrice(t *testing.T) {
	e := newTestEngine()
	now := time.Now()
	old := testOrder("old", "0x00000000000000000000000000000000000000a1", "-5", "500", now)
	newer := testOrder("new", "0x00000000000000000000000000000000000000a2", "-5", "500", now.Add(time.Second))
	buy := testOrder("buy", "0x00000000000000000000000000000000000000b1", "5", "-500", now.Add(2*time.Second))

	mustAddOrder(t, e, newer)
	mustAddOrder(t, e, old)
	mustAddOrder(t, e, buy)

	trades := pendingTrades(e)
	if len(trades) != 1 || trades[0].MakerOrder.OrderId != "old" {
		t.Fatalf("expected maker old, got %+v", trades)
	}
}
