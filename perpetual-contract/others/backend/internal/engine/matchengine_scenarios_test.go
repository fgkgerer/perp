// 撮合引擎场景测试（matchengine_scenarios_test.go）
//
// 覆盖 MatchEngine + OrderBook（价档+FIFO）的业务路径，与生产行为对齐：
//   - 新单主动吃簿（matchIncomingOrder）
//   - 定时交叉撮合（match / matchOrderBook）
//   - 撤单、过期、回滚、事件回放
//
// 订单字段约定（与链上一致）：
//   - paper > 0：做多（买单）；paper < 0：做空（卖单）
//   - 限价 = |credit| / |paper|（精度 1e18，测试中常用整数价如 100、110）
//   - 新进入且主动成交的为 taker；已在簿上的为 maker；成交价取 maker 价
//
// 运行：go test ./internal/engine/... -run TestScenario -v
package engine

import (
	"context"
	"math/big"
	"strconv"
	"testing"
	"time"

	"metanode/internal/config"
	"metanode/internal/model"
)

// --- 测试辅助函数 ---

// testOrderExp 构造带指定过期时间的订单（用于过期/剔除场景）。
func testOrderExp(id, signer, paper, credit string, created time.Time, exp time.Time) *model.Order {
	o := testOrder(id, signer, paper, credit, created)
	o.Expiration = exp.Unix()
	return o
}

// restOrder 不经撮合，直接把订单挂入内存簿（模拟 DB 恢复或测试定时交叉撮合的前置状态）。
func restOrder(t *testing.T, e *MatchEngine, order *model.Order) {
	t.Helper()
	e.addOrderWithRemain(order, absPaperString(order.PaperAmount))
}

func assertTradeCount(t *testing.T, trades []*MatchResult, want int) {
	t.Helper()
	if len(trades) != want {
		t.Fatalf("expected %d trades, got %d: %+v", want, len(trades), trades)
	}
}

// assertTrade 校验单笔成交的 taker/maker 角色与成交数量（paper 绝对值）。
func assertTrade(t *testing.T, tr *MatchResult, takerID, makerID, amount string) {
	t.Helper()
	if tr.TakerOrder.OrderId != takerID {
		t.Fatalf("taker want %s got %s", takerID, tr.TakerOrder.OrderId)
	}
	if tr.MakerOrder.OrderId != makerID {
		t.Fatalf("maker want %s got %s", makerID, tr.MakerOrder.OrderId)
	}
	if tr.MatchAmount != amount {
		t.Fatalf("amount want %s got %s", amount, tr.MatchAmount)
	}
}

func assertBookEmpty(t *testing.T, e *MatchEngine) {
	t.Helper()
	bids, asks := e.SnapshotOrderBook(testPerp, 20)
	if len(bids) != 0 || len(asks) != 0 {
		t.Fatalf("expected empty book, bids=%+v asks=%+v", bids, asks)
	}
}

// assertBidLevel 检查买盘第 idx 档（0=最优买价）的聚合数量。
func assertBidLevel(t *testing.T, e *MatchEngine, idx int, amount string) {
	t.Helper()
	bids, _ := e.SnapshotOrderBook(testPerp, 20)
	if len(bids) <= idx {
		t.Fatalf("expected bid level %d, only %d levels: %+v", idx, len(bids), bids)
	}
	if bids[idx].Amount != amount {
		t.Fatalf("bid[%d] amount want %s got %s", idx, amount, bids[idx].Amount)
	}
}

// assertAskLevel 检查卖盘第 idx 档（0=最优卖价）的聚合数量。
func assertAskLevel(t *testing.T, e *MatchEngine, idx int, amount string) {
	t.Helper()
	_, asks := e.SnapshotOrderBook(testPerp, 20)
	if len(asks) <= idx {
		t.Fatalf("expected ask level %d, only %d levels: %+v", idx, len(asks), asks)
	}
	if asks[idx].Amount != amount {
		t.Fatalf("ask[%d] amount want %s got %s", idx, amount, asks[idx].Amount)
	}
}

// --- 场景 1：价格不交叉 ---

// TestScenario_NoMatchWhenBuyPriceBelowBestAsk
// 卖一 100 先挂簿，买单限价 90 后进入：买价 < 卖价，不应成交，双边各自 resting。
func TestScenario_NoMatchWhenBuyPriceBelowBestAsk(t *testing.T) {
	e := newTestEngine()
	now := time.Now()
	mustAddOrder(t, e, testOrder("ask", "0x00000000000000000000000000000000000000a1", "-10", "1000", now)) // 卖 @100
	mustAddOrder(t, e, testOrder("bid", "0x00000000000000000000000000000000000000b1", "10", "-900", now))  // 买 @90

	assertTradeCount(t, pendingTrades(e), 0)
	assertBidLevel(t, e, 0, "10")
	assertAskLevel(t, e, 0, "10")
}

// --- 场景 2 & 3：部分成交 ---

// TestScenario_PartialFillTakerRestsRemainder
// 卖 5 @100，买 10 @100：成交 5，taker（买）剩余 5 挂入买簿。
func TestScenario_PartialFillTakerRestsRemainder(t *testing.T) {
	e := newTestEngine()
	now := time.Now()
	mustAddOrder(t, e, testOrder("ask", "0x00000000000000000000000000000000000000a1", "-5", "500", now))
	mustAddOrder(t, e, testOrder("buy", "0x00000000000000000000000000000000000000b1", "10", "-1000", now))

	trades := pendingTrades(e)
	assertTradeCount(t, trades, 1)
	assertTrade(t, trades[0], "buy", "ask", "5")
	assertBidLevel(t, e, 0, "5")
}

// TestScenario_PartialFillMakerRestsRemainder
// 卖 10 @100，买 5 @100：成交 5，maker（卖）剩余 5 继续挂在卖簿。
func TestScenario_PartialFillMakerRestsRemainder(t *testing.T) {
	e := newTestEngine()
	now := time.Now()
	mustAddOrder(t, e, testOrder("ask", "0x00000000000000000000000000000000000000a1", "-10", "1000", now))
	mustAddOrder(t, e, testOrder("buy", "0x00000000000000000000000000000000000000b1", "5", "-500", now))

	trades := pendingTrades(e)
	assertTradeCount(t, trades, 1)
	assertTrade(t, trades[0], "buy", "ask", "5")
	assertAskLevel(t, e, 0, "5")
}

// --- 场景 4：同价 FIFO，一笔 taker 连吃多 maker ---

// TestScenario_TakerEatsMultipleMakersSamePrice
// 三笔卖单同价 100（数量 3/4/3），一笔买 10：按 CreateTime 顺序连吃，产生 3 笔成交，簿清空。
func TestScenario_TakerEatsMultipleMakersSamePrice(t *testing.T) {
	e := newTestEngine()
	now := time.Now()
	mustAddOrder(t, e, testOrder("ask1", "0x00000000000000000000000000000000000000a1", "-3", "300", now))
	mustAddOrder(t, e, testOrder("ask2", "0x00000000000000000000000000000000000000a2", "-4", "400", now.Add(time.Millisecond)))
	mustAddOrder(t, e, testOrder("ask3", "0x00000000000000000000000000000000000000a3", "-3", "300", now.Add(2*time.Millisecond)))
	mustAddOrder(t, e, testOrder("buy", "0x00000000000000000000000000000000000000b1", "10", "-1000", now.Add(3*time.Millisecond)))

	trades := pendingTrades(e)
	assertTradeCount(t, trades, 3)
	assertTrade(t, trades[0], "buy", "ask1", "3")
	assertTrade(t, trades[1], "buy", "ask2", "4")
	assertTrade(t, trades[2], "buy", "ask3", "3")
	assertBookEmpty(t, e)
}

// --- 场景 5 & 6：多价档与 maker 价 ---

// TestScenario_TakerEatsBestPriceFirst
// 卖 110 与卖 100 同时在簿，买 5 @100 应优先吃掉更优卖价 100（ask-low），高价卖单仍在簿。
func TestScenario_TakerEatsBestPriceFirst(t *testing.T) {
	e := newTestEngine()
	now := time.Now()
	mustAddOrder(t, e, testOrder("ask-high", "0x00000000000000000000000000000000000000a1", "-5", "550", now)) // @110
	mustAddOrder(t, e, testOrder("ask-low", "0x00000000000000000000000000000000000000a2", "-5", "500", now))  // @100
	mustAddOrder(t, e, testOrder("buy", "0x00000000000000000000000000000000000000b1", "5", "-500", now))

	trades := pendingTrades(e)
	assertTradeCount(t, trades, 1)
	assertTrade(t, trades[0], "buy", "ask-low", "5")
	wantPrice := calculatePrice(testOrder("", "", "-5", "500", now)).String()
	if trades[0].MatchPrice != wantPrice {
		t.Fatalf("expected maker low price %s, got %s", wantPrice, trades[0].MatchPrice)
	}
	assertAskLevel(t, e, 0, "5") // ask-high 剩余
}

// TestScenario_MatchPriceIsMakerPrice
// maker 卖 @100，taker 买 @110（愿出更高价）：成交价仍为 maker 的 100，而非 taker 限价。
func TestScenario_MatchPriceIsMakerPrice(t *testing.T) {
	e := newTestEngine()
	now := time.Now()
	sell := testOrder("ask", "0x00000000000000000000000000000000000000a1", "-5", "500", now)
	mustAddOrder(t, e, sell)
	mustAddOrder(t, e, testOrder("buy", "0x00000000000000000000000000000000000000b1", "5", "-550", now))

	trades := pendingTrades(e)
	assertTradeCount(t, trades, 1)
	if trades[0].MatchPrice != calculatePrice(sell).String() {
		t.Fatalf("match price should be maker %s, got %s", calculatePrice(sell), trades[0].MatchPrice)
	}
}

// --- 场景 7 & 8：定时交叉撮合 match() ---

// TestScenario_PeriodicCrossMatchRestingOrders
// 买卖同价 100 已通过 restOrder 挂簿（未走 incoming 撮合），调用 match() 后应交叉成交。
func TestScenario_PeriodicCrossMatchRestingOrders(t *testing.T) {
	e := newTestEngine()
	now := time.Now()
	restOrder(t, e, testOrder("bid", "0x00000000000000000000000000000000000000b1", "10", "-1000", now))
	restOrder(t, e, testOrder("ask", "0x00000000000000000000000000000000000000a1", "-10", "1000", now))

	assertTradeCount(t, pendingTrades(e), 0)
	e.match()
	trades := pendingTrades(e)
	assertTradeCount(t, trades, 1)
	assertBookEmpty(t, e)
}

// TestScenario_PeriodicCrossMatchSkipsSelfTrade
// 同一 signer 同时挂买/卖：定时撮合必须跳过自成交，双边订单仍留在簿上。
func TestScenario_PeriodicCrossMatchSkipsSelfTrade(t *testing.T) {
	e := newTestEngine()
	now := time.Now()
	signer := "0x00000000000000000000000000000000000000a1"
	restOrder(t, e, testOrder("bid-self", signer, "5", "-500", now))
	restOrder(t, e, testOrder("ask-self", signer, "-5", "500", now))

	e.match()
	assertTradeCount(t, pendingTrades(e), 0)
	assertBidLevel(t, e, 0, "5")
	assertAskLevel(t, e, 0, "5")
}

// --- 场景 9 & 10：过期订单 ---

// TestScenario_ExpiredOrderNotResting
// expiration 已过的订单调用 matchIncomingOrder 时不应进入订单簿。
func TestScenario_ExpiredOrderNotResting(t *testing.T) {
	e := newTestEngine()
	now := time.Now()
	expired := testOrderExp("exp", "0x00000000000000000000000000000000000000b1", "10", "-1000", now, now.Add(-time.Minute))
	e.matchIncomingOrder(expired, absPaperString(expired.PaperAmount))
	assertBookEmpty(t, e)
}

// TestScenario_PruneExpiredOnMatch
// 已挂簿的过期单在 pruneExpired 时应被剔除，未过期单保留。
func TestScenario_PruneExpiredOnMatch(t *testing.T) {
	e := newTestEngine()
	now := time.Now()
	restOrder(t, e, testOrderExp("dead", "0x00000000000000000000000000000000000000a2", "-5", "500", now, now.Add(-time.Minute)))
	restOrder(t, e, testOrder("alive", "0x00000000000000000000000000000000000000a1", "-5", "500", now))

	book := e.getOrCreateBook(testPerp)
	book.mu.Lock()
	e.pruneExpired(book)
	book.mu.Unlock()

	_, asks := e.SnapshotOrderBook(testPerp, 10)
	if len(asks) != 1 || asks[0].Amount != "5" {
		t.Fatalf("expected only alive ask, got %+v", asks)
	}
}

// --- 场景 11 & 12：撤单与幂等 ---

// TestScenario_CancelPreventsLaterMatch
// 买单挂簿后撤单，后续卖单进入时不应与之成交。
func TestScenario_CancelPreventsLaterMatch(t *testing.T) {
	e := newTestEngine()
	now := time.Now()
	mustAddOrder(t, e, testOrder("bid", "0x00000000000000000000000000000000000000b1", "10", "-1000", now))
	if !e.RemoveOrder("bid") {
		t.Fatal("cancel failed")
	}
	mustAddOrder(t, e, testOrder("ask", "0x00000000000000000000000000000000000000a1", "-10", "1000", now))
	assertTradeCount(t, pendingTrades(e), 0)
}

// TestScenario_DuplicateOrderIdIgnored
// 相同 orderId 重复 AddOrder 不应叠加数量（簿内仅一份）。
func TestScenario_DuplicateOrderIdIgnored(t *testing.T) {
	e := newTestEngine()
	now := time.Now()
	buy := testOrder("dup", "0x00000000000000000000000000000000000000b1", "10", "-1000", now)
	mustAddOrder(t, e, buy)
	mustAddOrder(t, e, buy)
	assertBidLevel(t, e, 0, "10")
}

// --- 场景 13：链上失败回滚 ---

// TestScenario_RollbackRestoresBookRemain
// 模拟 match 已成交后 applyRollbackMatch：买卖双方剩余量应恢复到簿上。
func TestScenario_RollbackRestoresBookRemain(t *testing.T) {
	e := newTestEngine()
	now := time.Now()
	buy := testOrder("buy", "0x00000000000000000000000000000000000000b1", "10", "-1000", now)
	sell := testOrder("sell", "0x00000000000000000000000000000000000000a1", "-10", "1000", now)
	mustAddOrder(t, e, sell)
	mustAddOrder(t, e, buy)
	assertBookEmpty(t, e)

	trade := &MatchResult{
		MatchID:     "rb-1",
		TakerOrder:  buy,
		MakerOrder:  sell,
		MatchAmount: "10",
		MatchPrice:  calculatePrice(sell).String(),
	}
	matchAmt, _ := new(big.Int).SetString("10", 10)
	e.applyRollbackMatch(trade, matchAmt)

	assertBidLevel(t, e, 0, "10")
	assertAskLevel(t, e, 0, "10")
}

// --- 场景 14：WAL 事件回放 ---

// TestScenario_EventStreamOrderCancelReplay
// 事件序列为 order_accepted + order_canceled，新引擎回放后簿应为空。
func TestScenario_EventStreamOrderCancelReplay(t *testing.T) {
	store := newMemoryEngineEventStore()
	src := NewMatchEngine(config.MatchEngineConfig{}, nil, nil, store, nil, nil)
	src.setLeader(true)
	src.setRecovered(true)
	now := time.Now()
	buy := testOrder("ev-buy", "0x00000000000000000000000000000000000000b1", "10", "-1000", now)

	if err := src.AddOrder(buy); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Append(context.Background(), &model.EngineEvent{
		EventType: model.EngineEventOrderCanceled,
		OrderId:   "ev-buy",
	}); err != nil {
		t.Fatal(err)
	}

	replay := newTestEngine()
	events, _ := store.ListAfter(context.Background(), 0, 10)
	for _, ev := range events {
		if err := replay.applyEngineEvent(ev); err != nil {
			t.Fatal(err)
		}
	}
	assertBookEmpty(t, replay)
}

// --- 场景 15 & 16：深度与标记价 ---

// TestScenario_SnapshotRespectsLimit
// 5 个买价档，SnapshotOrderBook(limit=3) 只返回最优 3 档。
func TestScenario_SnapshotRespectsLimit(t *testing.T) {
	e := newTestEngine()
	now := time.Now()
	credits := []string{"-100", "-200", "-300", "-400", "-500"} // 买价 100~500
	for i, credit := range credits {
		mustAddOrder(t, e, testOrder("bid"+strconv.Itoa(i), "0x00000000000000000000000000000000000000b1", "1", credit, now.Add(time.Duration(i)*time.Millisecond)))
	}
	bids, _ := e.SnapshotOrderBook(testPerp, 3)
	if len(bids) != 3 {
		t.Fatalf("expected 3 bid levels, got %d", len(bids))
	}
}

// TestScenario_MidMarkPrice
// 买 100、卖 110 不交叉时，MidMarkPrice = (100+110)/2。
func TestScenario_MidMarkPrice(t *testing.T) {
	e := newTestEngine()
	now := time.Now()
	bid := testOrder("bid", "0x00000000000000000000000000000000000000b1", "10", "-1000", now)
	ask := testOrder("ask", "0x00000000000000000000000000000000000000a1", "-10", "1100", now)
	mustAddOrder(t, e, bid)
	mustAddOrder(t, e, ask)

	price, ok := e.MidMarkPrice(testPerp)
	if !ok {
		t.Fatal("expected mid price")
	}
	mid, _ := new(big.Int).SetString(price, 10)
	want := new(big.Int).Add(calculatePrice(bid), calculatePrice(ask))
	want.Div(want, big.NewInt(2))
	if mid.Cmp(want) != 0 {
		t.Fatalf("mid want %s got %s", want, mid)
	}
}

// TestScenario_MidMarkPriceSingleSide
// 仅有一边盘口时，标记价取该边最优价。
func TestScenario_MidMarkPriceSingleSide(t *testing.T) {
	e := newTestEngine()
	now := time.Now()
	bid := testOrder("bid", "0x00000000000000000000000000000000000000b1", "10", "-1000", now)
	mustAddOrder(t, e, bid)
	price, ok := e.MidMarkPrice(testPerp)
	if !ok {
		t.Fatal("expected bid-only mark")
	}
	if price != calculatePrice(bid).String() {
		t.Fatalf("unexpected mark %s", price)
	}
}

// --- 场景 17 & 18：卖 taker 多档与限价保护 ---

// TestScenario_SellDoesNotMatchBidBelowSellPrice
// 买 @90、卖 @100：卖 taker 限价 100，不会接受 90 的 bid，无成交。
func TestScenario_SellDoesNotMatchBidBelowSellPrice(t *testing.T) {
	e := newTestEngine()
	now := time.Now()
	mustAddOrder(t, e, testOrder("bid", "0x00000000000000000000000000000000000000b1", "4", "-360", now))   // 买 @90
	mustAddOrder(t, e, testOrder("sell", "0x00000000000000000000000000000000000000a1", "-7", "700", now)) // 卖 @100

	assertTradeCount(t, pendingTrades(e), 0)
	assertBidLevel(t, e, 0, "4")
	assertAskLevel(t, e, 0, "7")
}

// TestScenario_SellTakerWalksBidLevels
// 买 @110×3、买 @105×4，卖 @100×7：卖 taker 先吃高价买档，再吃次档，两笔成交后簿空。
func TestScenario_SellTakerWalksBidLevels(t *testing.T) {
	e := newTestEngine()
	now := time.Now()
	mustAddOrder(t, e, testOrder("bid-high", "0x00000000000000000000000000000000000000b1", "3", "-330", now)) // @110
	mustAddOrder(t, e, testOrder("bid-low", "0x00000000000000000000000000000000000000b2", "4", "-420", now))  // @105
	mustAddOrder(t, e, testOrder("sell", "0x00000000000000000000000000000000000000a1", "-7", "700", now))     // @100

	trades := pendingTrades(e)
	assertTradeCount(t, trades, 2)
	assertTrade(t, trades[0], "sell", "bid-high", "3")
	assertTrade(t, trades[1], "sell", "bid-low", "4")
	assertBookEmpty(t, e)
}
