package listener

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"metanode/internal/market"
	"metanode/internal/model"
	"metanode/internal/svc"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	defaultBinanceWsURL = "wss://stream.binance.com:9443/stream"
	defaultOkxWsURL     = "wss://ws.okx.com:8443/ws/v5/public"
	defaultCoinbaseWs   = "wss://ws-feed.exchange.coinbase.com"
	venueStaleAfter     = 120 * time.Second
)

type spotMarketBinding struct {
	perp            string
	marketName      string
	coinbaseProduct string
	binanceSymbol   string
	okxInstId       string
}

type venueTick struct {
	usd       *big.Float
	updatedAt time.Time
}

type perpSpotState struct {
	coinbase venueTick
	okx      venueTick
	binance  venueTick
	// 缓存加权结果
	price1e6  string
	display2  string
	updatedAt time.Time
}

// CompositeSpotIndex 订阅 Coinbase / OKX / Binance 现货 ticker，按 4:3:3 加权指数价。
type CompositeSpotIndex struct {
	svc     *svc.ServiceContext
	weights SpotWeights
	stop    chan struct{}
	once    sync.Once

	bindings []spotMarketBinding
	mu       sync.RWMutex
	byPerp   map[string]*perpSpotState
	lastSave map[string]time.Time
	saveMu   sync.Mutex
}

func NewCompositeSpotIndex(svc *svc.ServiceContext) *CompositeSpotIndex {
	cfg := svc.Config.SpotIndex
	w := SpotWeights{cfg.CoinbaseWeight, cfg.OkxWeight, cfg.BinanceWeight}
	var bindings []spotMarketBinding
	for _, m := range svc.Config.Markets {
		b := spotMarketBinding{
			perp:            strings.ToLower(strings.TrimSpace(m.Address)),
			marketName:      m.Name,
			coinbaseProduct: strings.TrimSpace(m.CoinbaseProduct),
			binanceSymbol:   strings.TrimSpace(m.BinanceSymbol),
			okxInstId:       strings.TrimSpace(m.OkxInstId),
		}
		if b.coinbaseProduct == "" && b.binanceSymbol == "" && b.okxInstId == "" {
			continue
		}
		bindings = append(bindings, b)
	}
	return &CompositeSpotIndex{
		svc:      svc,
		weights:  w,
		stop:     make(chan struct{}),
		bindings: bindings,
		byPerp:   make(map[string]*perpSpotState),
		lastSave: make(map[string]time.Time),
	}
}

func (c *CompositeSpotIndex) Start() {
	if !c.svc.Config.SpotIndex.Enabled {
		logx.Info("CompositeSpotIndex disabled")
		return
	}
	if len(c.bindings) == 0 {
		logx.Error("CompositeSpotIndex: no market symbols configured")
		return
	}
	logx.Infof("CompositeSpotIndex starting markets=%d weights=%d:%d:%d",
		len(c.bindings), c.weights.normalized().Coinbase, c.weights.normalized().Okx, c.weights.normalized().Binance)
	go c.runCoinbase()
	go c.runBinance()
	go c.runOkx()
}

func (c *CompositeSpotIndex) Stop() {
	c.once.Do(func() { close(c.stop) })
}

func (c *CompositeSpotIndex) IndexPrice1e6(perp string) (string, bool) {
	perp = strings.ToLower(strings.TrimSpace(perp))
	c.mu.RLock()
	defer c.mu.RUnlock()
	st, ok := c.byPerp[perp]
	if !ok || st.price1e6 == "" {
		return "", false
	}
	return st.price1e6, true
}

func (c *CompositeSpotIndex) IndexPriceDisplay(perp string) (string, bool) {
	perp = strings.ToLower(strings.TrimSpace(perp))
	c.mu.RLock()
	defer c.mu.RUnlock()
	st, ok := c.byPerp[perp]
	if !ok || st.display2 == "" {
		return "", false
	}
	return st.display2, true
}

func (c *CompositeSpotIndex) setVenue(perp, venue, priceUsd string) {
	f, ok := parseUsdFloat(priceUsd)
	if !ok {
		return
	}
	now := time.Now().UTC()
	c.mu.Lock()
	st, exists := c.byPerp[perp]
	if !exists {
		st = &perpSpotState{}
		c.byPerp[perp] = st
	}
	switch venue {
	case "coinbase":
		st.coinbase = venueTick{f, now}
	case "okx":
		st.okx = venueTick{f, now}
	case "binance":
		st.binance = venueTick{f, now}
	}
	c.recalcLocked(st)
	c.mu.Unlock()
}

func (c *CompositeSpotIndex) recalcLocked(st *perpSpotState) {
	now := time.Now().UTC()
	cb := activeVenuePrice(st.coinbase, now)
	okx := activeVenuePrice(st.okx, now)
	bn := activeVenuePrice(st.binance, now)
	out, ok := WeightedSpotUSD(cb, okx, bn, c.weights)
	if !ok {
		return
	}
	st.price1e6 = usdFloatTo1e6(out)
	st.display2 = FormatUsdDisplay2(out)
	st.updatedAt = now
}

func activeVenuePrice(v venueTick, now time.Time) *big.Float {
	if v.usd == nil || v.usd.Sign() <= 0 {
		return nil
	}
	if now.Sub(v.updatedAt) > venueStaleAfter {
		return nil
	}
	return v.usd
}

func (c *CompositeSpotIndex) maybeUpsertQuote(perp string) {
	if c.svc.MarketQuoteModel == nil {
		return
	}
	c.mu.RLock()
	st, ok := c.byPerp[perp]
	if !ok || st.display2 == "" {
		c.mu.RUnlock()
		return
	}
	display := st.display2
	name := ""
	for _, b := range c.bindings {
		if b.perp == perp {
			name = b.marketName
			break
		}
	}
	c.mu.RUnlock()

	if !c.shouldSave(perp) {
		return
	}
	q := &model.MarketQuote{
		Perp:       perp,
		MarketName: name,
		ProductID:  "spot-index",
		PriceUsd:   display,
		Source:     "composite-4-3-3",
		UpdatedAt:  time.Now().UTC(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.svc.MarketQuoteModel.Upsert(ctx, q); err != nil {
		logx.Errorf("CompositeSpotIndex upsert %s: %v", perp, err)
	}
	market.RecordSpotIndexPrice(ctx, c.svc, perp, display)
}

func (c *CompositeSpotIndex) shouldSave(perp string) bool {
	ms := c.throttleMs()
	c.saveMu.Lock()
	defer c.saveMu.Unlock()
	now := time.Now()
	if last, ok := c.lastSave[perp]; ok && now.Sub(last) < time.Duration(ms)*time.Millisecond {
		return false
	}
	c.lastSave[perp] = now
	return true
}

func (c *CompositeSpotIndex) throttleMs() int {
	if c.svc.Config.Coinbase.ThrottleMs > 0 {
		return c.svc.Config.Coinbase.ThrottleMs
	}
	return 500
}

func (c *CompositeSpotIndex) reconnectSec() int {
	if c.svc.Config.Coinbase.ReconnectSec > 0 {
		return c.svc.Config.Coinbase.ReconnectSec
	}
	return 5
}

// --- Coinbase ---

func (c *CompositeSpotIndex) runCoinbase() {
	products := make(map[string]spotMarketBinding)
	for _, b := range c.bindings {
		if b.coinbaseProduct != "" {
			products[b.coinbaseProduct] = b
		}
	}
	if len(products) == 0 {
		return
	}
	for {
		select {
		case <-c.stop:
			return
		default:
		}
		if err := c.connectCoinbase(products); err != nil {
			logx.Errorf("CompositeSpotIndex coinbase: %v", err)
		}
		select {
		case <-c.stop:
			return
		case <-time.After(time.Duration(c.reconnectSec()) * time.Second):
		}
	}
}

func (c *CompositeSpotIndex) connectCoinbase(products map[string]spotMarketBinding) error {
	wsURL := strings.TrimSpace(c.svc.Config.Coinbase.WsURL)
	if wsURL == "" {
		wsURL = defaultCoinbaseWs
	}
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	ids := make([]string, 0, len(products))
	for pid := range products {
		ids = append(ids, pid)
	}
	sub := coinbaseSubscribeMsg{
		Type: "subscribe",
		Channels: []struct {
			Name       string   `json:"name"`
			ProductIDs []string `json:"product_ids"`
		}{{Name: "ticker", ProductIDs: ids}},
	}
	if err := conn.WriteJSON(sub); err != nil {
		return err
	}
	logx.Infof("CompositeSpotIndex coinbase subscribed: %v", ids)
	for {
		select {
		case <-c.stop:
			return nil
		default:
		}
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		var msg coinbaseTickerMsg
		if err := json.Unmarshal(raw, &msg); err != nil || msg.Type != "ticker" {
			continue
		}
		b, ok := products[msg.ProductID]
		if !ok {
			continue
		}
		c.setVenue(b.perp, "coinbase", msg.Price)
		c.maybeUpsertQuote(b.perp)
	}
}

// --- Binance ---

type binanceCombinedMsg struct {
	Stream string          `json:"stream"`
	Data   binanceMiniData `json:"data"`
}

type binanceMiniData struct {
	Symbol    string `json:"s"`
	LastPrice string `json:"c"`
}

func (c *CompositeSpotIndex) runBinance() {
	symMap := make(map[string]spotMarketBinding)
	var streams []string
	for _, b := range c.bindings {
		sym := strings.ToLower(b.binanceSymbol)
		if sym == "" {
			continue
		}
		symMap[sym] = b
		streams = append(streams, sym+"@miniTicker")
	}
	if len(streams) == 0 {
		return
	}
	wsURL := strings.TrimSpace(c.svc.Config.SpotIndex.BinanceWsURL)
	if wsURL == "" {
		wsURL = defaultBinanceWsURL + "?streams=" + strings.Join(streams, "/")
	}
	for {
		select {
		case <-c.stop:
			return
		default:
		}
		if err := c.connectBinance(wsURL, symMap); err != nil {
			logx.Errorf("CompositeSpotIndex binance: %v", err)
		}
		select {
		case <-c.stop:
			return
		case <-time.After(time.Duration(c.reconnectSec()) * time.Second):
		}
	}
}

func (c *CompositeSpotIndex) connectBinance(wsURL string, symMap map[string]spotMarketBinding) error {
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	logx.Infof("CompositeSpotIndex binance connected: %s", wsURL)
	for {
		select {
		case <-c.stop:
			return nil
		default:
		}
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		var wrap binanceCombinedMsg
		if err := json.Unmarshal(raw, &wrap); err != nil {
			var direct binanceMiniData
			if err2 := json.Unmarshal(raw, &direct); err2 != nil {
				continue
			}
			wrap.Data = direct
		}
		sym := strings.ToLower(wrap.Data.Symbol)
		b, ok := symMap[sym]
		if !ok {
			continue
		}
		c.setVenue(b.perp, "binance", wrap.Data.LastPrice)
		c.maybeUpsertQuote(b.perp)
	}
}

// --- OKX ---

type okxSub struct {
	Op   string        `json:"op"`
	Args []okxSubEntry `json:"args"`
}

type okxSubEntry struct {
	Channel string `json:"channel"`
	InstID  string `json:"instId"`
}

type okxWsMsg struct {
	Arg  struct {
		Channel string `json:"channel"`
		InstID  string `json:"instId"`
	} `json:"arg"`
	Data []struct {
		Last string `json:"last"`
	} `json:"data"`
}

func (c *CompositeSpotIndex) runOkx() {
	instMap := make(map[string]spotMarketBinding)
	var args []okxSubEntry
	for _, b := range c.bindings {
		inst := b.okxInstId
		if inst == "" {
			continue
		}
		instMap[inst] = b
		args = append(args, okxSubEntry{Channel: "tickers", InstID: inst})
	}
	if len(args) == 0 {
		return
	}
	wsURL := strings.TrimSpace(c.svc.Config.SpotIndex.OkxWsURL)
	if wsURL == "" {
		wsURL = defaultOkxWsURL
	}
	for {
		select {
		case <-c.stop:
			return
		default:
		}
		if err := c.connectOkx(wsURL, instMap, args); err != nil {
			logx.Errorf("CompositeSpotIndex okx: %v", err)
		}
		select {
		case <-c.stop:
			return
		case <-time.After(time.Duration(c.reconnectSec()) * time.Second):
		}
	}
}

func (c *CompositeSpotIndex) connectOkx(wsURL string, instMap map[string]spotMarketBinding, args []okxSubEntry) error {
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := conn.WriteJSON(okxSub{Op: "subscribe", Args: args}); err != nil {
		return fmt.Errorf("okx subscribe: %w", err)
	}
	logx.Infof("CompositeSpotIndex okx subscribed: %d inst", len(args))
	for {
		select {
		case <-c.stop:
			return nil
		default:
		}
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		var msg okxWsMsg
		if err := json.Unmarshal(raw, &msg); err != nil || len(msg.Data) == 0 {
			continue
		}
		b, ok := instMap[msg.Arg.InstID]
		if !ok {
			continue
		}
		c.setVenue(b.perp, "okx", msg.Data[0].Last)
		c.maybeUpsertQuote(b.perp)
	}
}
