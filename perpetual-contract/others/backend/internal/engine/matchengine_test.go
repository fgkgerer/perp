// 撮合引擎单元测试（matchengine_test.go）
//
// 覆盖核心撮合语义与 Leader/Event 持久化路径，场景用例见 matchengine_scenarios_test.go。
package engine

import (
	"context"
	"testing"
	"time"

	"metanode/internal/config"
	"metanode/internal/model"
)

// testPerp 测试用永续合约地址（任意非空 hex 即可）。
const testPerp = "0x0000000000000000000000000000000000000001"

// newTestEngine 无 DB / 无 Redis / 无链的内存撮合引擎，AddOrder 同步入簿撮合。
func newTestEngine() *MatchEngine {
	return NewMatchEngine(config.MatchEngineConfig{
		MatchInterval:    100,
		BatchSize:        10,
		MaxPendingOrders: 100,
	}, nil, nil, nil, nil, nil)
}

// testOrder 构造测试订单。
// paper/credit 符号决定方向；CreateTime 影响同价 FIFO 排序。
func testOrder(id, signer, paper, credit string, created time.Time) *model.Order {
	return &model.Order{
		OrderId:      id,
		Perp:         testPerp,
		Signer:       signer,
		PaperAmount:  paper,
		CreditAmount: credit,
		MakerFeeRate: "0",
		TakerFeeRate: "0",
		Expiration:   time.Now().Add(time.Hour).Unix(),
		Nonce:        created.UnixNano(),
		Signature:    "0x",
		Status:       model.OrderStatusPending,
		FilledAmount: "0",
		CreateTime:   created,
		UpdateTime:   created,
	}
}

// pendingTrades 拷贝当前待上链成交队列（线程安全读取）。
func pendingTrades(e *MatchEngine) []*MatchResult {
	e.tradeMu.Lock()
	defer e.tradeMu.Unlock()
	out := make([]*MatchResult, len(e.pendingTrades))
	copy(out, e.pendingTrades)
	return out
}

// TestIncomingBuyIsTakerAgainstRestingSell
// 先挂卖后挂买：后到的买单为 taker，卖单为 maker，全量成交。
func TestIncomingBuyIsTakerAgainstRestingSell(t *testing.T) {
	e := newTestEngine()
	now := time.Now()
	sell := testOrder("sell-1", "0x00000000000000000000000000000000000000a1", "-10", "1000", now)
	buy := testOrder("buy-1", "0x00000000000000000000000000000000000000b1", "10", "-1000", now.Add(time.Second))

	mustAddOrder(t, e, sell)
	mustAddOrder(t, e, buy)

	trades := pendingTrades(e)
	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
	if trades[0].TakerOrder.OrderId != "buy-1" || trades[0].MakerOrder.OrderId != "sell-1" {
		t.Fatalf("wrong taker/maker: taker=%s maker=%s", trades[0].TakerOrder.OrderId, trades[0].MakerOrder.OrderId)
	}
	if trades[0].MatchAmount != "10" {
		t.Fatalf("wrong amount: %s", trades[0].MatchAmount)
	}
}

// TestIncomingSellIsTakerAgainstRestingBuy
// 先挂买后挂卖：卖单为 taker；成交价应为 maker（买）的价格。
func TestIncomingSellIsTakerAgainstRestingBuy(t *testing.T) {
	e := newTestEngine()
	now := time.Now()
	buy := testOrder("buy-1", "0x00000000000000000000000000000000000000b1", "10", "-1000", now)
	sell := testOrder("sell-1", "0x00000000000000000000000000000000000000a1", "-10", "1000", now.Add(time.Second))

	mustAddOrder(t, e, buy)
	mustAddOrder(t, e, sell)

	trades := pendingTrades(e)
	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
	if trades[0].TakerOrder.OrderId != "sell-1" || trades[0].MakerOrder.OrderId != "buy-1" {
		t.Fatalf("wrong taker/maker: taker=%s maker=%s", trades[0].TakerOrder.OrderId, trades[0].MakerOrder.OrderId)
	}
	if trades[0].MatchPrice != calculatePrice(buy).String() {
		t.Fatalf("expected maker price %s, got %s", calculatePrice(buy).String(), trades[0].MatchPrice)
	}
}

// TestSamePriceUsesTimePriority
// 同价两笔卖单：尽管 sell-new 先 add，sell-old 的 CreateTime 更早，应作为 maker 先成交。
func TestSamePriceUsesTimePriority(t *testing.T) {
	e := newTestEngine()
	now := time.Now()
	sellOld := testOrder("sell-old", "0x00000000000000000000000000000000000000a1", "-5", "500", now)
	sellNew := testOrder("sell-new", "0x00000000000000000000000000000000000000a2", "-5", "500", now.Add(time.Second))
	buy := testOrder("buy-1", "0x00000000000000000000000000000000000000b1", "5", "-500", now.Add(2*time.Second))

	mustAddOrder(t, e, sellNew)
	mustAddOrder(t, e, sellOld)
	mustAddOrder(t, e, buy)

	trades := pendingTrades(e)
	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
	if trades[0].MakerOrder.OrderId != "sell-old" {
		t.Fatalf("expected old order first, got %s", trades[0].MakerOrder.OrderId)
	}
}

// TestSelfMatchIsSkipped
// 同一 signer 的卖单在队首时跳过，与第三方卖单成交。
func TestSelfMatchIsSkipped(t *testing.T) {
	e := newTestEngine()
	now := time.Now()
	sameSignerSell := testOrder("sell-self", "0x00000000000000000000000000000000000000a1", "-5", "450", now)
	otherSell := testOrder("sell-other", "0x00000000000000000000000000000000000000a2", "-5", "500", now.Add(time.Second))
	buy := testOrder("buy-1", "0x00000000000000000000000000000000000000a1", "5", "-500", now.Add(2*time.Second))

	mustAddOrder(t, e, sameSignerSell)
	mustAddOrder(t, e, otherSell)
	mustAddOrder(t, e, buy)

	trades := pendingTrades(e)
	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
	if trades[0].MakerOrder.OrderId != "sell-other" {
		t.Fatalf("expected self-match skip to use sell-other, got %s", trades[0].MakerOrder.OrderId)
	}
}

// TestRemoveOrderRemovesResidualFromBook
// RemoveOrder 后深度应为空（价档+索引结构下 O(1) 撤单）。
func TestRemoveOrderRemovesResidualFromBook(t *testing.T) {
	e := newTestEngine()
	now := time.Now()
	buy := testOrder("buy-1", "0x00000000000000000000000000000000000000b1", "10", "-1000", now)

	mustAddOrder(t, e, buy)
	if !e.RemoveOrder("buy-1") {
		t.Fatal("expected order to be removed")
	}
	bids, asks := e.SnapshotOrderBook(testPerp, 10)
	if len(bids) != 0 || len(asks) != 0 {
		t.Fatalf("expected empty book, got bids=%d asks=%d", len(bids), len(asks))
	}
}

// TestOrderAcceptedEventCanBeReplayed
// 单机回放 order_accepted 事件后，订单应出现在簿上。
func TestOrderAcceptedEventCanBeReplayed(t *testing.T) {
	store := newMemoryEngineEventStore()
	e := NewMatchEngine(config.MatchEngineConfig{}, nil, nil, store, nil, nil)
	now := time.Now()
	buy := testOrder("buy-1", "0x00000000000000000000000000000000000000b1", "10", "-1000", now)

	if _, err := e.appendOrderEvent(context.Background(), buy, absPaperString(buy.PaperAmount)); err != nil {
		t.Fatalf("append order event: %v", err)
	}
	events, err := store.ListAfter(context.Background(), 0, 10)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}

	replay := newTestEngine()
	if err := replay.applyEngineEvent(events[0]); err != nil {
		t.Fatalf("apply event: %v", err)
	}
	bids, _ := replay.SnapshotOrderBook(testPerp, 10)
	if len(bids) != 1 || bids[0].Amount != "10" {
		t.Fatalf("expected replayed bid amount 10, got %+v", bids)
	}
}

// TestAddOrderWithEventStoreOnlyMutatesBookAfterEventConsumption
// 生产路径：AddOrder 只写 WAL，Leader 消费 event 后才改内存簿。
func TestAddOrderWithEventStoreOnlyMutatesBookAfterEventConsumption(t *testing.T) {
	store := newMemoryEngineEventStore()
	e := NewMatchEngine(config.MatchEngineConfig{}, nil, nil, store, nil, nil)
	e.setLeader(true)
	e.setRecovered(true)
	now := time.Now()
	buy := testOrder("buy-1", "0x00000000000000000000000000000000000000b1", "10", "-1000", now)

	if err := e.AddOrder(buy); err != nil {
		t.Fatalf("AddOrder: %v", err)
	}
	bids, _ := e.SnapshotOrderBook(testPerp, 10)
	if len(bids) != 0 {
		t.Fatalf("expected AddOrder not to mutate book directly, got %+v", bids)
	}

	e.pollEngineEvents(context.Background())
	e.match()
	bids, _ = e.SnapshotOrderBook(testPerp, 10)
	if len(bids) != 1 || bids[0].Amount != "10" {
		t.Fatalf("expected event consumption to populate book, got %+v", bids)
	}
}

// TestUnresolvedMatchRestoresPendingTradeAndReservation
// 重启恢复：未 settled 的 match_enqueued 应回到 pendingTrades，且簿内 remain 已预扣。
func TestUnresolvedMatchRestoresPendingTradeAndReservation(t *testing.T) {
	store := newMemoryEngineEventStore()
	now := time.Now()
	buy := testOrder("buy-1", "0x00000000000000000000000000000000000000b1", "10", "-1000", now)
	sell := testOrder("sell-1", "0x00000000000000000000000000000000000000a1", "-10", "1000", now.Add(time.Second))
	source := NewMatchEngine(config.MatchEngineConfig{}, nil, nil, store, nil, nil)
	trade := &MatchResult{
		MatchID:     "match-1",
		TakerOrder:  sell,
		MakerOrder:  buy,
		MatchAmount: "10",
		MatchPrice:  calculatePrice(buy).String(),
	}
	if _, err := source.appendMatchEvent(context.Background(), model.EngineEventMatchEnqueued, trade); err != nil {
		t.Fatalf("append match event: %v", err)
	}

	replay := NewMatchEngine(config.MatchEngineConfig{}, nil, nil, store, nil, nil)
	replay.restoreOrder(buy, absPaperString(buy.PaperAmount))
	replay.restoreOrder(sell, absPaperString(sell.PaperAmount))
	replay.restoreUnresolvedMatches(context.Background())

	trades := pendingTrades(replay)
	if len(trades) != 1 || trades[0].MatchID != "match-1" {
		t.Fatalf("expected restored pending match, got %+v", trades)
	}
	bids, asks := replay.SnapshotOrderBook(testPerp, 10)
	if len(bids) != 0 || len(asks) != 0 {
		t.Fatalf("expected reserved orders removed from book, got bids=%+v asks=%+v", bids, asks)
	}
}

// mustAddOrder 测试用下单入口（无 eventModel 时同步撮合）。
func mustAddOrder(t *testing.T, e *MatchEngine, order *model.Order) {
	t.Helper()
	if err := e.AddOrder(order); err != nil {
		t.Fatalf("AddOrder(%s): %v", order.OrderId, err)
	}
}

// memoryEngineEventStore 内存版 EngineEvent 存储，用于 WAL 回放测试。
type memoryEngineEventStore struct {
	events []*model.EngineEvent
}

func newMemoryEngineEventStore() *memoryEngineEventStore {
	return &memoryEngineEventStore{}
}

func (s *memoryEngineEventStore) Append(_ context.Context, event *model.EngineEvent) (int64, error) {
	cp := *event
	cp.Seq = int64(len(s.events) + 1)
	s.events = append(s.events, &cp)
	event.Seq = cp.Seq
	return cp.Seq, nil
}

func (s *memoryEngineEventStore) LatestSeq(_ context.Context) (int64, error) {
	return int64(len(s.events)), nil
}

func (s *memoryEngineEventStore) ListAfter(_ context.Context, seq int64, limit int) ([]*model.EngineEvent, error) {
	var out []*model.EngineEvent
	for _, ev := range s.events {
		if ev.Seq > seq {
			out = append(out, ev)
		}
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

// FindUnresolvedMatches 返回已 enqueue 但未 settled/rollback 的成交事件。
func (s *memoryEngineEventStore) FindUnresolvedMatches(_ context.Context, limit int) ([]*model.EngineEvent, error) {
	done := make(map[string]bool)
	for _, ev := range s.events {
		if ev.MatchId != "" && (ev.EventType == model.EngineEventMatchSettled || ev.EventType == model.EngineEventMatchRollback) {
			done[ev.MatchId] = true
		}
	}
	var out []*model.EngineEvent
	for _, ev := range s.events {
		if ev.EventType == model.EngineEventMatchEnqueued && ev.MatchId != "" && !done[ev.MatchId] {
			out = append(out, ev)
		}
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}
