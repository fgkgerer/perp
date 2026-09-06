package listener

import (
	"math/big"
	"strings"
)

// 默认现货指数权重：Coinbase : OKX : Binance = 4 : 3 : 3
const (
	DefaultCoinbaseWeight = 4
	DefaultOkxWeight      = 3
	DefaultBinanceWeight  = 3
)

// IndexPriceProvider 现货指数（供 API、资金费 Keeper 使用）。
type IndexPriceProvider interface {
	IndexPrice1e6(perp string) (price string, ok bool)
	IndexPriceDisplay(perp string) (price string, ok bool)
}

// SpotWeights 三所权重，和为 0 时回退默认 4:3:3。
type SpotWeights struct {
	Coinbase int
	Okx      int
	Binance  int
}

// Normalized 权重和为 0 时回退 4:3:3。
func (w SpotWeights) Normalized() SpotWeights {
	if w.Coinbase+w.Okx+w.Binance > 0 {
		return w
	}
	return SpotWeights{DefaultCoinbaseWeight, DefaultOkxWeight, DefaultBinanceWeight}
}

func (w SpotWeights) normalized() SpotWeights { return w.Normalized() }

// WeightedSpotUSD 按权重对有效价格加权平均（仅含 price>0 的交易所；权重按比例重归一）。
func WeightedSpotUSD(cb, okx, binance *big.Float, w SpotWeights) (*big.Float, bool) {
	w = w.normalized()
	type part struct {
		v *big.Float
		w int
	}
	var parts []part
	if cb != nil && cb.Sign() > 0 {
		parts = append(parts, part{cb, w.Coinbase})
	}
	if okx != nil && okx.Sign() > 0 {
		parts = append(parts, part{okx, w.Okx})
	}
	if binance != nil && binance.Sign() > 0 {
		parts = append(parts, part{binance, w.Binance})
	}
	if len(parts) == 0 {
		return nil, false
	}
	sumW := 0
	for _, p := range parts {
		sumW += p.w
	}
	acc := new(big.Float)
	for _, p := range parts {
		term := new(big.Float).Mul(p.v, big.NewFloat(float64(p.w)))
		acc.Add(acc, term)
	}
	acc.Quo(acc, big.NewFloat(float64(sumW)))
	return acc, true
}

func parseUsdFloat(s string) (*big.Float, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, false
	}
	f, ok := new(big.Float).SetString(s)
	return f, ok && f.Sign() > 0
}

func usdTo1e6(usd string) string {
	f, ok := parseUsdFloat(usd)
	if !ok {
		return "0"
	}
	scale := new(big.Float).SetFloat64(1e6)
	out, _ := new(big.Float).Mul(f, scale).Int(nil)
	if out == nil {
		return "0"
	}
	return out.String()
}

func usdFloatTo1e6(f *big.Float) string {
	if f == nil {
		return "0"
	}
	scale := new(big.Float).SetFloat64(1e6)
	out, _ := new(big.Float).Mul(f, scale).Int(nil)
	if out == nil {
		return "0"
	}
	return out.String()
}

// FormatUsdDisplay2 格式化为保留 2 位小数的美元字符串（前端展示）。
func FormatUsdDisplay2(usd *big.Float) string {
	if usd == nil {
		return "0.00"
	}
	return usd.Text('f', 2)
}

// FormatIndexDisplayFrom1e6 将 1e6 精度整数串格式化为 2 位小数。
func FormatIndexDisplayFrom1e6(price1e6 string) string {
	z := new(big.Int)
	if _, ok := z.SetString(strings.TrimSpace(price1e6), 10); !ok {
		return "0.00"
	}
	f := new(big.Float).SetInt(z)
	f.Quo(f, big.NewFloat(1e6))
	return FormatUsdDisplay2(f)
}
