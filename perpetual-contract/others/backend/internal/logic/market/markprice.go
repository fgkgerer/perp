package market

import (
	"context"
	"math/big"
	"strings"

	"metanode/internal/listener"
	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/ethereum/go-ethereum/common"
)

// OpenQuoteResult 开仓参考价（前端直接用于展示与 EIP-712 限价）。
type OpenQuoteResult struct {
	PriceUsd   string
	PriceRaw   string
	Source     string
	BestBid    string
	BestAsk    string
	MarkPrice  string
	IndexPrice string
}

// ResolveMarkPrice 展示用合约价：优先订单簿（双边 mid / 单边最优价），空簿时读链上 getMarkPrice。
func ResolveMarkPrice(ctx context.Context, svcCtx *svc.ServiceContext, perpAddress string) string {
	if svcCtx.OrderBook != nil {
		if raw, ok := svcCtx.OrderBook.MidMarkPrice(perpAddress); ok {
			return FormatMarkDisplay(raw)
		}
	}
	if svcCtx.Chain != nil {
		perp := common.HexToAddress(perpAddress)
		if mp, err := svcCtx.Chain.GetMarkPrice(ctx, perp); err == nil && mp.Sign() > 0 {
			return FormatMarkDisplay(mp.String())
		}
	}
	return ""
}

func applyMarkPrice(pm *types.PerpMarket, ctx context.Context, svcCtx *svc.ServiceContext, perpAddress string) {
	if v := ResolveMarkPrice(ctx, svcCtx, perpAddress); v != "" {
		pm.MarkPrice = v
	}
}

// ResolveOpenQuote 开仓系统价：做多取卖一（吃 ask），做空取买一（吃 bid）；无盘口再 mid → 链 → 指数。
func ResolveOpenQuote(ctx context.Context, svcCtx *svc.ServiceContext, perpAddress, side string) OpenQuoteResult {
	out := OpenQuoteResult{
		MarkPrice:  ResolveMarkPrice(ctx, svcCtx, perpAddress),
		IndexPrice: resolveIndexDisplay(svcCtx, perpAddress),
	}
	side = strings.ToLower(strings.TrimSpace(side))

	var bidRaw, askRaw string
	if svcCtx.OrderBook != nil {
		bids, asks := svcCtx.OrderBook.SnapshotOrderBook(perpAddress, 1)
		if len(bids) > 0 {
			bidRaw = bids[0].Price
			out.BestBid = FormatMarkDisplay(bidRaw)
		}
		if len(asks) > 0 {
			askRaw = asks[0].Price
			out.BestAsk = FormatMarkDisplay(askRaw)
		}
		switch side {
		case "long":
			if askRaw != "" {
				out.PriceRaw = askRaw
				out.PriceUsd = out.BestAsk
				out.Source = "best_ask"
				return out
			}
		case "short":
			if bidRaw != "" {
				out.PriceRaw = bidRaw
				out.PriceUsd = out.BestBid
				out.Source = "best_bid"
				return out
			}
		}
		if raw, ok := svcCtx.OrderBook.MidMarkPrice(perpAddress); ok {
			out.PriceRaw = raw
			out.PriceUsd = FormatMarkDisplay(raw)
			out.Source = "orderbook_mid"
			return out
		}
	}

	if svcCtx.Chain != nil {
		perp := common.HexToAddress(perpAddress)
		if mp, err := svcCtx.Chain.GetMarkPrice(ctx, perp); err == nil && mp.Sign() > 0 {
			out.PriceRaw = mp.String()
			out.PriceUsd = FormatMarkDisplay(mp.String())
			out.Source = "chain_mark"
			return out
		}
	}
	if out.IndexPrice != "" && out.IndexPrice != "0.00" {
		out.PriceUsd = out.IndexPrice
		out.PriceRaw = indexDisplayTo1e6(out.IndexPrice)
		out.Source = "spot_index"
	}
	return out
}

func resolveIndexDisplay(svcCtx *svc.ServiceContext, perp string) string {
	if svcCtx.IndexPrice == nil {
		return ""
	}
	if d, ok := svcCtx.IndexPrice.IndexPriceDisplay(perp); ok {
		return d
	}
	if raw, ok := svcCtx.IndexPrice.IndexPrice1e6(perp); ok {
		return listener.FormatIndexDisplayFrom1e6(raw)
	}
	return ""
}

func indexDisplayTo1e6(display string) string {
	f, ok := new(big.Float).SetString(strings.TrimSpace(display))
	if !ok {
		return "0"
	}
	out, _ := new(big.Float).Mul(f, big.NewFloat(1e6)).Int(nil)
	if out == nil {
		return "0"
	}
	return out.String()
}

// MarkIndex1e6ForFunding 资金费溢价计算用 mark/index（统一 1e6 整数，与 Keeper 指数口径一致）。
func MarkIndex1e6ForFunding(ctx context.Context, svcCtx *svc.ServiceContext, perpAddress string) (mark, index *big.Int) {
	mark = big.NewInt(0)
	index = big.NewInt(0)
	if display := ResolveMarkPrice(ctx, svcCtx, perpAddress); display != "" && display != "0.00" {
		if z, ok := new(big.Int).SetString(indexDisplayTo1e6(display), 10); ok {
			mark = z
		}
	}
	if svcCtx.IndexPrice != nil {
		if px, ok := svcCtx.IndexPrice.IndexPrice1e6(perpAddress); ok {
			if z, ok := new(big.Int).SetString(strings.TrimSpace(px), 10); ok && z.Sign() > 0 {
				index = z
			}
		}
	}
	if index.Sign() == 0 && mark.Sign() > 0 {
		index = new(big.Int).Set(mark)
	}
	if mark.Sign() == 0 && svcCtx.Chain != nil {
		perp := common.HexToAddress(perpAddress)
		if mp, err := svcCtx.Chain.GetMarkPrice(ctx, perp); err == nil && mp.Sign() > 0 {
			mark = normalizePriceTo1e6(mp)
		}
	}
	return mark, index
}

func normalizePriceTo1e6(z *big.Int) *big.Int {
	if z == nil || z.Sign() <= 0 {
		return big.NewInt(0)
	}
	threshold := new(big.Int).Exp(big.NewInt(10), big.NewInt(15), nil)
	if z.Cmp(threshold) >= 0 {
		return new(big.Int).Div(z, big.NewInt(1e12)) // 1e18 -> 1e6
	}
	return new(big.Int).Set(z)
}

// FormatMarkDisplay 将订单簿价（多为 credit 口径）或链上 mark 格式化为 2 位小数 USD 字符串。
func FormatMarkDisplay(priceRaw string) string {
	z := new(big.Int)
	if _, ok := z.SetString(priceRaw, 10); !ok || z.Sign() <= 0 {
		return "0.00"
	}
	// >= 1e15 视为 1e18 精度（Oracle）；否则视为与 USDC credit 一致的 1e6 口径（订单簿 / 测试网 mark）
	threshold := new(big.Int).Exp(big.NewInt(10), big.NewInt(15), nil)
	f := new(big.Float).SetInt(z)
	if z.Cmp(threshold) >= 0 {
		f.Quo(f, big.NewFloat(1e18))
	} else {
		f.Quo(f, big.NewFloat(1e6))
	}
	return listener.FormatUsdDisplay2(f)
}
