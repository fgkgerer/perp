package engine

// 撮合引擎：内存订单簿 + 周期性调用 chain.BuildMatchTradeData / Perpetual.trade。
// 链上角色：私钥地址须为 Dealer.validOrderSender（见 SubmitPerpTrade）。

import (
	"context"
	"math/big"
	"sort"
	"strings"
	"sync"
	"time"

	"metanode/internal/chain"
	"metanode/internal/config"
	"metanode/internal/model"
	"metanode/internal/svc"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// MatchEngine 内存撮合 +（可选）链上结算。约定：买单为 taker、卖单为 maker，与 Trading._matchOrders 一致。
type MatchEngine struct {
	config     config.MatchEngineConfig
	orderModel model.OrderModel
	tradeModel model.TradeModel
	chain      *chain.Client

	// 订单簿：perp -> side -> orders
	orderBooks map[string]*OrderBook
	mu         sync.RWMutex

	// orderId -> 剩余可撮合 paper 绝对值（与链上累计成交量配合；订单对象仍保留签名时的完整 paper/credit）
	remain map[string]*big.Int
	remMu  sync.Mutex

	// 待提交的交易
	pendingTrades []*MatchResult
	tradeMu       sync.Mutex

	stopCh chan struct{}
}

// OrderBook 订单簿
type OrderBook struct {
	Perp       string
	BuyOrders  []*model.Order // 买单（做多），按价格从高到低排序
	SellOrders []*model.Order // 卖单（做空），按价格从低到高排序
	mu         sync.RWMutex
}

// MatchResult 撮合结果
type MatchResult struct {
	TakerOrder  *model.Order
	MakerOrder  *model.Order
	MatchAmount string // 成交数量（paper 绝对值）
	MatchPrice  string // 成交价格
}

// NewMatchEngine 创建撮合引擎
func NewMatchEngine(cfg config.MatchEngineConfig, orderModel model.OrderModel, tradeModel model.TradeModel, ch *chain.Client) *MatchEngine {
	return &MatchEngine{
		config:     cfg,
		orderModel: orderModel,
		tradeModel: tradeModel,
		chain:      ch,
		orderBooks: make(map[string]*OrderBook),
		remain:     make(map[string]*big.Int),
		stopCh:     make(chan struct{}),
	}
}

// Start 启动撮合引擎；perpAddresses 用于从 DB 恢复未成交订单。
func (e *MatchEngine) Start(perpAddresses []string) {
	logx.Info("Match engine starting...")
	e.RestorePendingOrders(context.Background(), perpAddresses)
	go e.matchLoop()
	go e.submitLoop()
}

// RestorePendingOrders 重启后从 DB 加载 pending / partial 订单回内存簿。
func (e *MatchEngine) RestorePendingOrders(ctx context.Context, perpAddresses []string) {
	if e.orderModel == nil {
		logx.Error("RestorePendingOrders skipped: order model nil")
		return
	}
	limit := e.config.MaxPendingOrders
	if limit <= 0 {
		limit = 10000
	}
	var restored int
	for _, perp := range perpAddresses {
		perp = strings.TrimSpace(perp)
		if perp == "" {
			continue
		}
		orders, err := e.orderModel.FindPendingOrders(ctx, perp, limit)
		if err != nil {
			logx.Errorf("RestorePendingOrders perp=%s: %v", perp, err)
			continue
		}
		for _, o := range orders {
			full := absPaperString(o.PaperAmount)
			filled, _ := new(big.Int).SetString(o.FilledAmount, 10)
			rem := new(big.Int).Sub(full, filled)
			if rem.Sign() <= 0 {
				continue
			}
			e.restoreOrder(o, rem)
			restored++
		}
	}
	if restored > 0 {
		logx.Infof("Match engine restored %d pending orders from DB", restored)
	}
}

// Stop 停止撮合引擎
func (e *MatchEngine) Stop() {
	close(e.stopCh)
}

func absPaperString(paper string) *big.Int {
	z := new(big.Int)
	z.SetString(paper, 10)
	z.Abs(z)
	return z
}

func orderExpired(o *model.Order, now int64) bool {
	if o == nil {
		return true
	}
	return o.Expiration <= now
}

func (e *MatchEngine) dropExpiredLocked(book *OrderBook, side string, idx int, o *model.Order) {
	switch side {
	case "buy":
		book.BuyOrders = append(book.BuyOrders[:idx], book.BuyOrders[idx+1:]...)
	case "sell":
		book.SellOrders = append(book.SellOrders[:idx], book.SellOrders[idx+1:]...)
	}
	e.remMu.Lock()
	delete(e.remain, o.OrderId)
	e.remMu.Unlock()
	if e.orderModel != nil {
		_ = e.orderModel.UpdateStatus(context.Background(), o.OrderId, model.OrderStatusCancelled, o.FilledAmount)
	}
	logx.Infof("expired order removed from book: %s expiration=%d", o.OrderId, o.Expiration)
}

// pruneExpiredAtHead 移除队首过期单，避免反复链上 revert。
func (e *MatchEngine) pruneExpiredAtHead(book *OrderBook) {
	now := time.Now().Unix()
	for len(book.BuyOrders) > 0 && orderExpired(book.BuyOrders[0], now) {
		e.dropExpiredLocked(book, "buy", 0, book.BuyOrders[0])
	}
	for len(book.SellOrders) > 0 && orderExpired(book.SellOrders[0], now) {
		e.dropExpiredLocked(book, "sell", 0, book.SellOrders[0])
	}
}

// AddOrder 添加订单到订单簿
func (e *MatchEngine) AddOrder(order *model.Order) {
	e.addOrderWithRemain(order, absPaperString(order.PaperAmount))
}

func (e *MatchEngine) restoreOrder(order *model.Order, remain *big.Int) {
	if remain == nil || remain.Sign() <= 0 {
		return
	}
	e.addOrderWithRemain(order, new(big.Int).Set(remain))
}

func (e *MatchEngine) addOrderWithRemain(order *model.Order, remain *big.Int) {
	e.remMu.Lock()
	e.remain[order.OrderId] = new(big.Int).Set(remain)
	e.remMu.Unlock()

	e.mu.Lock()
	defer e.mu.Unlock()

	orderBook, ok := e.orderBooks[order.Perp]
	if !ok {
		orderBook = &OrderBook{Perp: order.Perp}
		e.orderBooks[order.Perp] = orderBook
	}

	orderBook.mu.Lock()
	defer orderBook.mu.Unlock()

	if e.orderInBook(orderBook, order.OrderId) {
		return
	}

	paperAmount := new(big.Int)
	paperAmount.SetString(order.PaperAmount, 10)

	if paperAmount.Sign() > 0 {
		orderBook.BuyOrders = append(orderBook.BuyOrders, order)
		sort.Slice(orderBook.BuyOrders, func(i, j int) bool {
			priceI := calculatePrice(orderBook.BuyOrders[i])
			priceJ := calculatePrice(orderBook.BuyOrders[j])
			return priceI.Cmp(priceJ) > 0
		})
	} else {
		orderBook.SellOrders = append(orderBook.SellOrders, order)
		sort.Slice(orderBook.SellOrders, func(i, j int) bool {
			priceI := calculatePrice(orderBook.SellOrders[i])
			priceJ := calculatePrice(orderBook.SellOrders[j])
			return priceI.Cmp(priceJ) < 0
		})
	}
}

func (e *MatchEngine) orderInBook(book *OrderBook, orderID string) bool {
	for _, o := range book.BuyOrders {
		if o.OrderId == orderID {
			return true
		}
	}
	for _, o := range book.SellOrders {
		if o.OrderId == orderID {
			return true
		}
	}
	return false
}

// matchLoop 撮合循环
func (e *MatchEngine) matchLoop() {
	ticker := time.NewTicker(time.Duration(e.config.MatchInterval) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-e.stopCh:
			return
		case <-ticker.C:
			e.match()
		}
	}
}

// match 执行撮合
func (e *MatchEngine) match() {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, orderBook := range e.orderBooks {
		e.matchOrderBook(orderBook)
	}
}

// matchOrderBook 撮合单个订单簿
func (e *MatchEngine) matchOrderBook(book *OrderBook) {
	book.mu.Lock()
	defer book.mu.Unlock()

	for {
		e.pruneExpiredAtHead(book)
		if len(book.BuyOrders) == 0 || len(book.SellOrders) == 0 {
			break
		}

		buyOrder := book.BuyOrders[0]
		sellOrder := book.SellOrders[0]

		buyPrice := calculatePrice(buyOrder)
		sellPrice := calculatePrice(sellOrder)

		if buyPrice.Cmp(sellPrice) < 0 {
			break
		}

		e.remMu.Lock()
		buyRem := new(big.Int).Set(e.remain[buyOrder.OrderId])
		sellRem := new(big.Int).Set(e.remain[sellOrder.OrderId])
		e.remMu.Unlock()

		matchAmount := new(big.Int)
		if buyRem.Cmp(sellRem) < 0 {
			matchAmount.Set(buyRem)
		} else {
			matchAmount.Set(sellRem)
		}
		if matchAmount.Sign() == 0 {
			break
		}

		result := &MatchResult{
			TakerOrder:  buyOrder,
			MakerOrder:  sellOrder,
			MatchAmount: matchAmount.String(),
			MatchPrice:  sellPrice.String(),
		}

		e.tradeMu.Lock()
		e.pendingTrades = append(e.pendingTrades, result)
		e.tradeMu.Unlock()

		e.remMu.Lock()
		e.remain[buyOrder.OrderId].Sub(e.remain[buyOrder.OrderId], matchAmount)
		e.remain[sellOrder.OrderId].Sub(e.remain[sellOrder.OrderId], matchAmount)
		if e.remain[buyOrder.OrderId].Sign() == 0 {
			book.BuyOrders = book.BuyOrders[1:]
			delete(e.remain, buyOrder.OrderId)
		}
		if e.remain[sellOrder.OrderId].Sign() == 0 {
			book.SellOrders = book.SellOrders[1:]
			delete(e.remain, sellOrder.OrderId)
		}
		e.remMu.Unlock()
	}
}

// submitLoop 提交交易循环
func (e *MatchEngine) submitLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-e.stopCh:
			return
		case <-ticker.C:
			e.submitTrades()
		}
	}
}

// submitTrades 提交交易到链上
func (e *MatchEngine) submitTrades() {
	e.tradeMu.Lock()
	if len(e.pendingTrades) == 0 {
		e.tradeMu.Unlock()
		return
	}

	trades := e.pendingTrades
	e.pendingTrades = nil
	e.tradeMu.Unlock()

	batch := make([]*MatchResult, 0, e.config.BatchSize)
	for _, trade := range trades {
		batch = append(batch, trade)
		if len(batch) >= e.config.BatchSize {
			e.submitBatch(batch)
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		e.submitBatch(batch)
	}
}

// submitBatch 对每笔撮合：编码 tradeData → Perpetual.trade → 等待收据；链上成功才更新 DB。
func (e *MatchEngine) submitBatch(trades []*MatchResult) {
	if e.orderModel == nil || e.tradeModel == nil {
		logx.Error("order/trade model nil, skip DB updates")
		return
	}
	ctx := context.Background()
	for _, trade := range trades {
		matchAmt, ok := new(big.Int).SetString(trade.MatchAmount, 10)
		if !ok {
			logx.Errorf("bad match amount %s", trade.MatchAmount)
			continue
		}

		if e.chain == nil {
			logx.Error("chain client nil, rollback match (no on-chain settle)")
			e.rollbackMatch(trade, matchAmt)
			continue
		}

		now := time.Now().Unix()
		if orderExpired(trade.TakerOrder, now) || orderExpired(trade.MakerOrder, now) {
			logx.Errorf("skip on-chain trade: order expired taker=%s maker=%s", trade.TakerOrder.OrderId, trade.MakerOrder.OrderId)
			e.rollbackMatch(trade, matchAmt)
			continue
		}

		td, err := chain.BuildMatchTradeData(trade.TakerOrder, trade.MakerOrder, matchAmt)
		if err != nil {
			logx.Errorf("build trade data: %v", err)
			e.rollbackMatch(trade, matchAmt)
			continue
		}

		perp := common.HexToAddress(trade.TakerOrder.Perp)
		tx, err := e.chain.SubmitPerpTrade(ctx, perp, td)
		if err != nil {
			logx.Errorf("SubmitPerpTrade failed: %v", err)
			e.rollbackMatch(trade, matchAmt)
			continue
		}

		txHash := tx.Hash().Hex()
		rec, err := bind.WaitMined(ctx, e.chain.RPC(), tx)
		if err != nil {
			logx.Errorf("WaitMined %s: %v", txHash, err)
			e.rollbackMatch(trade, matchAmt)
			continue
		}
		if rec == nil || rec.Status != 1 {
			logx.Errorf("trade tx reverted or failed: %s status=%v", txHash, recStatus(rec))
			e.rollbackMatch(trade, matchAmt)
			continue
		}

		blockNum := int64(rec.BlockNumber.Uint64())
		e.applyFillStatus(ctx, trade.TakerOrder.OrderId, trade.TakerOrder.PaperAmount, matchAmt.String())
		e.applyFillStatus(ctx, trade.MakerOrder.OrderId, trade.MakerOrder.PaperAmount, matchAmt.String())

		if _, err := e.tradeModel.Insert(ctx, &model.Trade{
			TradeId:      generateTradeId(),
			Perp:         trade.TakerOrder.Perp,
			TakerOrderId: trade.TakerOrder.OrderId,
			MakerOrderId: trade.MakerOrder.OrderId,
			Taker:        trade.TakerOrder.Signer,
			Maker:        trade.MakerOrder.Signer,
			PaperAmount:  trade.MatchAmount,
			Price:        trade.MatchPrice,
			TxHash:       txHash,
			BlockNumber:  blockNum,
			CreateTime:   time.Now(),
		}); err != nil {
			logx.Errorf("insert trade: %v", err)
		} else {
			logx.Infof("trade settled on-chain: tx=%s perp=%s amount=%s", txHash, trade.TakerOrder.Perp, matchAmt.String())
		}
	}
}

func recStatus(rec *types.Receipt) uint64 {
	if rec == nil {
		return 0
	}
	return rec.Status
}

func (e *MatchEngine) rollbackMatch(trade *MatchResult, matchAmt *big.Int) {
	if trade == nil || matchAmt == nil || matchAmt.Sign() <= 0 {
		return
	}
	e.remMu.Lock()
	for _, oid := range []string{trade.TakerOrder.OrderId, trade.MakerOrder.OrderId} {
		if e.remain[oid] == nil {
			e.remain[oid] = big.NewInt(0)
		}
		e.remain[oid].Add(e.remain[oid], matchAmt)
	}
	e.remMu.Unlock()

	e.mu.Lock()
	defer e.mu.Unlock()
	book, ok := e.orderBooks[trade.TakerOrder.Perp]
	if !ok {
		book = &OrderBook{Perp: trade.TakerOrder.Perp}
		e.orderBooks[trade.TakerOrder.Perp] = book
	}
	book.mu.Lock()
	defer book.mu.Unlock()

	if !e.orderInBook(book, trade.TakerOrder.OrderId) {
		e.reinsertOrderLocked(book, trade.TakerOrder)
	}
	if !e.orderInBook(book, trade.MakerOrder.OrderId) {
		e.reinsertOrderLocked(book, trade.MakerOrder)
	}
	logx.Infof("match rolled back in memory: taker=%s maker=%s amount=%s", trade.TakerOrder.OrderId, trade.MakerOrder.OrderId, matchAmt.String())
}

func (e *MatchEngine) reinsertOrderLocked(book *OrderBook, order *model.Order) {
	paperAmount := new(big.Int)
	paperAmount.SetString(order.PaperAmount, 10)
	if paperAmount.Sign() > 0 {
		book.BuyOrders = append(book.BuyOrders, order)
		sort.Slice(book.BuyOrders, func(i, j int) bool {
			return calculatePrice(book.BuyOrders[i]).Cmp(calculatePrice(book.BuyOrders[j])) > 0
		})
	} else {
		book.SellOrders = append(book.SellOrders, order)
		sort.Slice(book.SellOrders, func(i, j int) bool {
			return calculatePrice(book.SellOrders[i]).Cmp(calculatePrice(book.SellOrders[j])) < 0
		})
	}
}

func (e *MatchEngine) applyFillStatus(ctx context.Context, orderID, signedPaper, deltaStr string) {
	// 按订单原始签名数量累计 filled；达到全额则 Status=Filled，否则 PartialFill。
	o, err := e.orderModel.FindOne(ctx, orderID)
	if err != nil || o == nil {
		return
	}
	full := absPaperString(signedPaper)
	delta, _ := new(big.Int).SetString(deltaStr, 10)
	prev, _ := new(big.Int).SetString(o.FilledAmount, 10)
	total := new(big.Int).Add(prev, delta)
	status := model.OrderStatusPartialFill
	if total.Cmp(full) >= 0 {
		status = model.OrderStatusFilled
		total.Set(full)
	}
	_ = e.orderModel.UpdateStatus(ctx, orderID, status, total.String())
}

// calculatePrice 计算订单价格 |credit|/|paper|（1e18 精度）
func calculatePrice(order *model.Order) *big.Int {
	paper := new(big.Int)
	paper.SetString(order.PaperAmount, 10)
	paper.Abs(paper)

	credit := new(big.Int)
	credit.SetString(order.CreditAmount, 10)
	credit.Abs(credit)

	if paper.Sign() == 0 {
		return big.NewInt(0)
	}

	precision := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	price := new(big.Int).Mul(credit, precision)
	price.Div(price, paper)
	return price
}

func generateTradeId() string {
	return time.Now().Format("20060102150405") + randomString(8)
}

// SnapshotOrderBook 返回内存订单簿聚合深度（price 为 |credit|/|paper| 的 1e18 精度字符串）。
func (e *MatchEngine) SnapshotOrderBook(perp string, limit int) (bids, asks []svc.OrderBookLevel) {
	if limit <= 0 {
		limit = 20
	}
	perp = strings.TrimSpace(perp)
	if perp == "" {
		return nil, nil
	}

	e.mu.RLock()
	book, ok := e.orderBooks[perp]
	e.mu.RUnlock()
	if !ok || book == nil {
		return nil, nil
	}

	book.mu.RLock()
	defer book.mu.RUnlock()

	e.remMu.Lock()
	defer e.remMu.Unlock()

	bidMap := aggregateBookLevels(book.BuyOrders, e.remain)
	askMap := aggregateBookLevels(book.SellOrders, e.remain)

	bids = levelsFromMap(bidMap, limit, true)
	asks = levelsFromMap(askMap, limit, false)
	return bids, asks
}

func aggregateBookLevels(orders []*model.Order, remain map[string]*big.Int) map[string]*big.Int {
	out := make(map[string]*big.Int)
	for _, o := range orders {
		if o == nil {
			continue
		}
		rem := remain[o.OrderId]
		if rem == nil || rem.Sign() <= 0 {
			continue
		}
		price := calculatePrice(o).String()
		if out[price] == nil {
			out[price] = new(big.Int)
		}
		out[price].Add(out[price], rem)
	}
	return out
}

func levelsFromMap(m map[string]*big.Int, limit int, desc bool) []svc.OrderBookLevel {
	if len(m) == 0 {
		return nil
	}
	prices := make([]string, 0, len(m))
	for p := range m {
		prices = append(prices, p)
	}
	sort.Slice(prices, func(i, j int) bool {
		pi, _ := new(big.Int).SetString(prices[i], 10)
		pj, _ := new(big.Int).SetString(prices[j], 10)
		if desc {
			return pi.Cmp(pj) > 0
		}
		return pi.Cmp(pj) < 0
	})
	if len(prices) > limit {
		prices = prices[:limit]
	}
	out := make([]svc.OrderBookLevel, 0, len(prices))
	for _, p := range prices {
		out = append(out, svc.OrderBookLevel{
			Price:  p,
			Amount: m[p].String(),
		})
	}
	return out
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
