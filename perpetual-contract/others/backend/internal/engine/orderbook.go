package engine

import (
	"container/list"
	"math/big"
	"strings"
	"sync"

	"metanode/internal/model"
	"metanode/internal/svc"
)

// OrderBook 价档 + FIFO 队列订单簿（买单价高优先，卖单价低优先，同价时间优先）。
type OrderBook struct {
	Perp    string
	bids    *bookSide
	asks    *bookSide
	entries map[string]*bookEntry
	mu      sync.RWMutex
}

type bookEntry struct {
	order  *model.Order
	remain *big.Int
	price  *big.Int
	side   *bookSide
	level  *priceLevel
	elem   *list.Element // 在 level.queue 中的位置
}

type bookSide struct {
	isBuy  bool
	book   *OrderBook
	levels map[string]*priceLevel // priceKey -> level
	head   *priceLevel            // 最优价档（买=最高价，卖=最低价）
}

type priceLevel struct {
	priceKey string
	price    *big.Int
	totalRem *big.Int
	queue    *list.List // FIFO: 元素为 *bookEntry
	prev     *priceLevel
	next     *priceLevel
}

func newOrderBook(perp string) *OrderBook {
	b := &OrderBook{
		Perp:    perp,
		entries: make(map[string]*bookEntry),
	}
	b.bids = &bookSide{isBuy: true, book: b, levels: make(map[string]*priceLevel)}
	b.asks = &bookSide{isBuy: false, book: b, levels: make(map[string]*priceLevel)}
	return b
}

func (b *OrderBook) hasOrder(orderID string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.hasOrderLocked(orderID)
}

func (b *OrderBook) hasOrderLocked(orderID string) bool {
	_, ok := b.entries[orderID]
	return ok
}

func (b *OrderBook) remainingOf(orderID string) *big.Int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.remainingOfLocked(orderID)
}

func (b *OrderBook) remainingOfLocked(orderID string) *big.Int {
	e := b.entries[orderID]
	if e == nil || e.remain == nil {
		return big.NewInt(0)
	}
	return new(big.Int).Set(e.remain)
}

func (b *OrderBook) insertResting(order *model.Order, remain *big.Int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.insertRestingLocked(order, remain)
}

func (b *OrderBook) insertRestingLocked(order *model.Order, remain *big.Int) {
	if order == nil || remain == nil || remain.Sign() <= 0 {
		return
	}
	price := calculatePrice(order)
	if price.Sign() <= 0 {
		return
	}
	if _, ok := b.entries[order.OrderId]; ok {
		return
	}

	side := b.asks
	if parseBigInt(order.PaperAmount).Sign() > 0 {
		side = b.bids
	}

	entry := &bookEntry{
		order:  order,
		remain: new(big.Int).Set(remain),
		price:  new(big.Int).Set(price),
		side:   side,
	}
	side.attach(entry)
	b.entries[order.OrderId] = entry
}

func (b *OrderBook) remove(orderID string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.removeLocked(orderID)
}

func (b *OrderBook) removeLocked(orderID string) bool {
	entry := b.entries[orderID]
	if entry == nil {
		return false
	}
	entry.side.detach(entry)
	delete(b.entries, orderID)
	return true
}

func (b *OrderBook) applyFill(orderID string, amount *big.Int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.applyFillLocked(orderID, amount)
}

func (b *OrderBook) applyFillLocked(orderID string, amount *big.Int) {
	if amount == nil || amount.Sign() <= 0 {
		return
	}
	entry := b.entries[orderID]
	if entry == nil || entry.remain == nil {
		return
	}
	entry.remain.Sub(entry.remain, amount)
	if entry.level != nil && entry.level.totalRem != nil {
		entry.level.totalRem.Sub(entry.level.totalRem, amount)
		if entry.level.totalRem.Sign() < 0 {
			entry.level.totalRem.SetInt64(0)
		}
	}
	if entry.remain.Sign() <= 0 {
		entry.side.detach(entry)
		delete(b.entries, orderID)
	}
}

func (b *OrderBook) addRemain(orderID string, amount *big.Int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.addRemainLocked(orderID, amount)
}

func (b *OrderBook) addRemainLocked(orderID string, amount *big.Int) {
	if amount == nil || amount.Sign() <= 0 {
		return
	}
	entry := b.entries[orderID]
	if entry == nil {
		return
	}
	entry.remain.Add(entry.remain, amount)
	if entry.level != nil && entry.level.totalRem != nil {
		entry.level.totalRem.Add(entry.level.totalRem, amount)
	}
}

func (side *bookSide) attach(entry *bookEntry) {
	key := entry.price.String()
	lvl := side.levels[key]
	if lvl == nil {
		lvl = &priceLevel{
			priceKey: key,
			price:    new(big.Int).Set(entry.price),
			totalRem: new(big.Int),
			queue:    list.New(),
		}
		side.levels[key] = lvl
		side.insertLevel(lvl)
	}
	entry.level = lvl
	side.enqueue(entry, lvl)
	lvl.totalRem.Add(lvl.totalRem, entry.remain)
}

func entryTimeBefore(a, b *bookEntry) bool {
	if !a.order.CreateTime.Equal(b.order.CreateTime) {
		return a.order.CreateTime.Before(b.order.CreateTime)
	}
	if a.order.Id != b.order.Id {
		return a.order.Id < b.order.Id
	}
	return a.order.OrderId < b.order.OrderId
}

func (side *bookSide) enqueue(entry *bookEntry, lvl *priceLevel) {
	var inserted *list.Element
	for e := lvl.queue.Front(); e != nil; e = e.Next() {
		existing := e.Value.(*bookEntry)
		if entryTimeBefore(entry, existing) {
			inserted = lvl.queue.InsertBefore(entry, e)
			break
		}
	}
	if inserted == nil {
		inserted = lvl.queue.PushBack(entry)
	}
	entry.elem = inserted
}

func (side *bookSide) detach(entry *bookEntry) {
	if entry == nil || entry.level == nil || entry.elem == nil {
		return
	}
	lvl := entry.level
	if entry.remain != nil && lvl.totalRem != nil {
		lvl.totalRem.Sub(lvl.totalRem, entry.remain)
		if lvl.totalRem.Sign() < 0 {
			lvl.totalRem.SetInt64(0)
		}
	}
	lvl.queue.Remove(entry.elem)
	entry.elem = nil
	entry.level = nil

	if lvl.queue.Len() == 0 {
		side.removeLevel(lvl)
	}
}

func (side *bookSide) insertLevel(lvl *priceLevel) {
	if side.head == nil {
		side.head = lvl
		return
	}
	if side.isBetter(lvl.price, side.head.price) {
		lvl.next = side.head
		side.head.prev = lvl
		side.head = lvl
		return
	}
	cur := side.head
	for cur.next != nil && !side.isBetter(lvl.price, cur.next.price) {
		cur = cur.next
	}
	lvl.prev = cur
	lvl.next = cur.next
	if cur.next != nil {
		cur.next.prev = lvl
	}
	cur.next = lvl
}

func (side *bookSide) removeLevel(lvl *priceLevel) {
	delete(side.levels, lvl.priceKey)
	if side.head == lvl {
		side.head = lvl.next
		if side.head != nil {
			side.head.prev = nil
		}
		return
	}
	if lvl.prev != nil {
		lvl.prev.next = lvl.next
	}
	if lvl.next != nil {
		lvl.next.prev = lvl.prev
	}
}

// isBetter 返回 a 是否比 b 更优（更靠近盘口）。
func (side *bookSide) isBetter(a, b *big.Int) bool {
	if side.isBuy {
		return a.Cmp(b) > 0
	}
	return a.Cmp(b) < 0
}

func (side *bookSide) priceCrosses(takerPrice *big.Int, makerPrice *big.Int, takerIsBuy bool) bool {
	if takerIsBuy {
		return makerPrice.Cmp(takerPrice) <= 0
	}
	return makerPrice.Cmp(takerPrice) >= 0
}

func (b *OrderBook) nextMatchableMaker(taker *model.Order, takerPrice *big.Int, takerIsBuy bool) (*model.Order, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.nextMatchableMakerLocked(taker, takerPrice, takerIsBuy)
}

func (b *OrderBook) nextMatchableMakerLocked(taker *model.Order, takerPrice *big.Int, takerIsBuy bool) (*model.Order, bool) {
	var side *bookSide
	if takerIsBuy {
		side = b.asks
	} else {
		side = b.bids
	}
	if side.head == nil {
		return nil, false
	}

	for lvl := side.head; lvl != nil; lvl = lvl.next {
		if !side.priceCrosses(takerPrice, lvl.price, takerIsBuy) {
			return nil, false
		}
		for e := lvl.queue.Front(); e != nil; e = e.Next() {
			entry := e.Value.(*bookEntry)
			if entry == nil || entry.order == nil || entry.remain.Sign() <= 0 {
				continue
			}
			if strings.EqualFold(entry.order.Signer, taker.Signer) {
				continue
			}
			return entry.order, true
		}
	}
	return nil, false
}

func (b *OrderBook) bestBidAsk() (bestBid, bestAsk *bookEntry) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.bestBidAskLocked()
}

func (b *OrderBook) bestBidAskLocked() (bestBid, bestAsk *bookEntry) {
	if b.bids.head != nil {
		for e := b.bids.head.queue.Front(); e != nil; e = e.Next() {
			entry := e.Value.(*bookEntry)
			if entry != nil && entry.remain.Sign() > 0 {
				bestBid = entry
				break
			}
		}
	}
	if b.asks.head != nil {
		for e := b.asks.head.queue.Front(); e != nil; e = e.Next() {
			entry := e.Value.(*bookEntry)
			if entry != nil && entry.remain.Sign() > 0 {
				bestAsk = entry
				break
			}
		}
	}
	return bestBid, bestAsk
}

func (b *OrderBook) bestCrossPair() (bidEntry, askEntry *bookEntry, ok bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.bestCrossPairLocked()
}

func (b *OrderBook) bestCrossPairLocked() (bidEntry, askEntry *bookEntry, ok bool) {
	bid := b.bids.frontEntryLocked("")
	if bid == nil {
		return nil, nil, false
	}
	ask := b.asks.frontEntryLocked(bid.order.Signer)
	if ask == nil {
		ask = b.asks.frontEntryLocked("")
	}
	if ask == nil {
		return nil, nil, false
	}
	if bid.price.Cmp(ask.price) < 0 {
		return nil, nil, false
	}
	if strings.EqualFold(bid.order.Signer, ask.order.Signer) {
		return nil, nil, false
	}
	return bid, ask, true
}

func (side *bookSide) frontEntryLocked(skipSigner string) *bookEntry {
	for lvl := side.head; lvl != nil; lvl = lvl.next {
		for e := lvl.queue.Front(); e != nil; e = e.Next() {
			entry := e.Value.(*bookEntry)
			if entry == nil || entry.order == nil || entry.remain.Sign() <= 0 {
				continue
			}
			if skipSigner != "" && strings.EqualFold(entry.order.Signer, skipSigner) {
				continue
			}
			return entry
		}
	}
	return nil
}

func (b *OrderBook) pruneExpired(now int64, onExpired func(*model.Order)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.pruneExpiredLocked(now, onExpired)
}

func (b *OrderBook) pruneExpiredLocked(now int64, onExpired func(*model.Order)) {
	var expired []*bookEntry
	for _, entry := range b.entries {
		if entry != nil && entry.order != nil && orderExpired(entry.order, now) {
			expired = append(expired, entry)
		}
	}
	for _, entry := range expired {
		if onExpired != nil {
			onExpired(entry.order)
		}
		entry.side.detach(entry)
		delete(b.entries, entry.order.OrderId)
	}
}

func (b *OrderBook) snapshotBids(limit int) []svc.OrderBookLevel {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.bids.snapshotLevels(limit)
}

func (b *OrderBook) snapshotAsks(limit int) []svc.OrderBookLevel {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.asks.snapshotLevels(limit)
}

func (side *bookSide) snapshotLevels(limit int) []svc.OrderBookLevel {
	if limit <= 0 || side.head == nil {
		return nil
	}
	out := make([]svc.OrderBookLevel, 0, limit)
	for lvl := side.head; lvl != nil && len(out) < limit; lvl = lvl.next {
		if lvl.totalRem == nil || lvl.totalRem.Sign() <= 0 {
			continue
		}
		out = append(out, svc.OrderBookLevel{
			Price:  lvl.priceKey,
			Amount: lvl.totalRem.String(),
		})
	}
	return out
}
