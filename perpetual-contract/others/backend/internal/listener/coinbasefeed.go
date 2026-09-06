package listener

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"metanode/internal/model"
	"metanode/internal/svc"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
)

const defaultCoinbaseWsURL = "wss://ws-feed.exchange.coinbase.com"

type coinbaseSubscribeMsg struct {
	Type     string `json:"type"`
	Channels []struct {
		Name       string   `json:"name"`
		ProductIDs []string `json:"product_ids"`
	} `json:"channels"`
}

type coinbaseTickerMsg struct {
	Type      string `json:"type"`
	ProductID string `json:"product_id"`
	Price     string `json:"price"`
	Open24h   string `json:"open_24h"`
	Volume24h string `json:"volume_24h"`
	Low24h    string `json:"low_24h"`
	High24h   string `json:"high_24h"`
}

// CoinbaseFeed 订阅 Coinbase ticker，UPSERT market_quotes 供 Supabase Realtime 推送。
type CoinbaseFeed struct {
	svc      *svc.ServiceContext
	stop     chan struct{}
	once     sync.Once
	products map[string]productMeta
	cache    map[string]*model.MarketQuote
	cacheMu  sync.RWMutex
	lastSave map[string]time.Time
	saveMu   sync.Mutex
}

type productMeta struct {
	perp       string
	marketName string
}

func NewCoinbaseFeed(svc *svc.ServiceContext) *CoinbaseFeed {
	products := make(map[string]productMeta)
	for _, m := range svc.Config.Markets {
		pid := strings.TrimSpace(m.CoinbaseProduct)
		if pid == "" {
			continue
		}
		products[pid] = productMeta{
			perp:       strings.ToLower(strings.TrimSpace(m.Address)),
			marketName: m.Name,
		}
	}
	return &CoinbaseFeed{
		svc:      svc,
		stop:     make(chan struct{}),
		products: products,
		cache:    make(map[string]*model.MarketQuote),
		lastSave: make(map[string]time.Time),
	}
}

func (f *CoinbaseFeed) Start() {
	cfg := f.svc.Config.Coinbase
	if !cfg.Enabled {
		logx.Info("Coinbase feed disabled")
		return
	}
	if len(f.products) == 0 {
		logx.Error("Coinbase feed skipped: no CoinbaseProduct in Markets config")
		return
	}
	if f.svc.MarketQuoteModel == nil {
		logx.Error("Coinbase feed skipped: MarketQuoteModel nil (check Supabase.DataSource)")
		return
	}
	logx.Infof("Coinbase feed starting: products=%d ws=%s", len(f.products), f.wsURL())
	go f.run()
}

func (f *CoinbaseFeed) Stop() {
	f.once.Do(func() { close(f.stop) })
}

// IndexPriceDisplay 返回 Coinbase 现货价（2 位小数），无缓存时 ok=false。
func (f *CoinbaseFeed) IndexPriceDisplay(perp string) (price string, ok bool) {
	perp = strings.ToLower(strings.TrimSpace(perp))
	f.cacheMu.RLock()
	defer f.cacheMu.RUnlock()
	for _, q := range f.cache {
		if q != nil && strings.EqualFold(q.Perp, perp) && q.PriceUsd != "" {
			if px, ok := parseUsdFloat(q.PriceUsd); ok {
				return FormatUsdDisplay2(px), true
			}
		}
	}
	return "", false
}

// IndexPrice1e6 返回 Coinbase 指数价（6 位 USDC 精度字符串），无缓存时 ok=false。
func (f *CoinbaseFeed) IndexPrice1e6(perp string) (price string, ok bool) {
	perp = strings.ToLower(strings.TrimSpace(perp))
	f.cacheMu.RLock()
	defer f.cacheMu.RUnlock()
	for _, q := range f.cache {
		if q != nil && strings.EqualFold(q.Perp, perp) && q.PriceUsd != "" {
			return usdTo1e6(q.PriceUsd), true
		}
	}
	return "", false
}

func (f *CoinbaseFeed) run() {
	for {
		select {
		case <-f.stop:
			return
		default:
		}
		if err := f.connectOnce(); err != nil {
			logx.Errorf("Coinbase feed: %v", err)
		}
		select {
		case <-f.stop:
			return
		case <-time.After(time.Duration(f.reconnectSec()) * time.Second):
		}
	}
}

func (f *CoinbaseFeed) connectOnce() error {
	wsURL := f.wsURL()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial %s: %w", wsURL, err)
	}
	defer conn.Close()

	ids := make([]string, 0, len(f.products))
	for pid := range f.products {
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
		return fmt.Errorf("subscribe: %w", err)
	}
	logx.Infof("Coinbase feed subscribed: %v", ids)

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-f.stop:
				return
			case <-ticker.C:
				_ = conn.WriteMessage(websocket.PingMessage, nil)
			}
		}
	}()
	defer close(done)

	for {
		select {
		case <-f.stop:
			return nil
		default:
		}
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}
		var msg coinbaseTickerMsg
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}
		if msg.Type != "ticker" {
			continue
		}
		f.handleTicker(&msg)
	}
}

func (f *CoinbaseFeed) handleTicker(msg *coinbaseTickerMsg) {
	meta, ok := f.products[msg.ProductID]
	if !ok {
		return
	}
	change, changePct := calcChange(msg.Price, msg.Open24h)
	q := &model.MarketQuote{
		Perp:                  meta.perp,
		MarketName:            meta.marketName,
		ProductID:             msg.ProductID,
		PriceUsd:              strings.TrimSpace(msg.Price),
		Open24h:               strings.TrimSpace(msg.Open24h),
		Volume24h:             strings.TrimSpace(msg.Volume24h),
		Low24h:                strings.TrimSpace(msg.Low24h),
		High24h:               strings.TrimSpace(msg.High24h),
		PriceChange24h:        change,
		PriceChangePercent24h: changePct,
		Source:                "coinbase",
		UpdatedAt:             time.Now().UTC(),
	}

	f.cacheMu.Lock()
	f.cache[msg.ProductID] = q
	f.cacheMu.Unlock()

	if !f.shouldSave(meta.perp) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := f.svc.MarketQuoteModel.Upsert(ctx, q); err != nil {
		logx.Errorf("Coinbase feed upsert %s: %v", meta.perp, err)
	}
}

func (f *CoinbaseFeed) shouldSave(perp string) bool {
	f.saveMu.Lock()
	defer f.saveMu.Unlock()
	now := time.Now()
	if last, ok := f.lastSave[perp]; ok && now.Sub(last) < time.Duration(f.throttleMs())*time.Millisecond {
		return false
	}
	f.lastSave[perp] = now
	return true
}

func (f *CoinbaseFeed) wsURL() string {
	u := strings.TrimSpace(f.svc.Config.Coinbase.WsURL)
	if u == "" {
		return defaultCoinbaseWsURL
	}
	return u
}

func (f *CoinbaseFeed) throttleMs() int {
	n := f.svc.Config.Coinbase.ThrottleMs
	if n <= 0 {
		return 500
	}
	return n
}

func (f *CoinbaseFeed) reconnectSec() int {
	n := f.svc.Config.Coinbase.ReconnectSec
	if n <= 0 {
		return 5
	}
	return n
}

func calcChange(price, open string) (change, pct string) {
	p, ok1 := new(big.Float).SetString(strings.TrimSpace(price))
	o, ok2 := new(big.Float).SetString(strings.TrimSpace(open))
	if !ok1 || !ok2 || o.Sign() == 0 {
		return "0", "0"
	}
	diff := new(big.Float).Sub(p, o)
	change = diff.Text('f', 8)
	ratio := new(big.Float).Quo(diff, o)
	ratio.Mul(ratio, big.NewFloat(100))
	pct = ratio.Text('f', 4)
	return change, pct
}

