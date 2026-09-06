package engine

import (
	"math/big"
	"strings"
)

// SimulatedFillLevel 模拟撮合单档成交。
type SimulatedFillLevel struct {
	PriceRaw    string
	AmountPaper string
}

// FillSimulation 按当前内存簿模拟 taker 吃单（不修改簿）。
type FillSimulation struct {
	FilledPaper   *big.Int
	UnfilledPaper *big.Int
	AvgPriceRaw   *big.Int // 加权均价（与 calculatePrice 同口径）
	WorstPriceRaw *big.Int // 买=最高成交价，卖=最低成交价
	Levels        []SimulatedFillLevel
}

// SimulateTakerFill 模拟 taker 按限价吃对手盘；sizePaper 为 paper 绝对值（1e18）。
func (b *OrderBook) SimulateTakerFill(takerIsBuy bool, sizePaper, limitPrice *big.Int, skipSigner string) FillSimulation {
	out := FillSimulation{
		FilledPaper:   big.NewInt(0),
		UnfilledPaper: big.NewInt(0),
	}
	if b == nil || sizePaper == nil || sizePaper.Sign() <= 0 || limitPrice == nil || limitPrice.Sign() <= 0 {
		if sizePaper != nil && sizePaper.Sign() > 0 {
			out.UnfilledPaper = new(big.Int).Set(sizePaper)
		}
		return out
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	side := b.asks
	if !takerIsBuy {
		side = b.bids
	}
	if side == nil || side.head == nil {
		out.UnfilledPaper = new(big.Int).Set(sizePaper)
		return out
	}

	remaining := new(big.Int).Set(sizePaper)
	notional := big.NewInt(0) // sum(priceRaw * fillPaper) for weighted avg
	worst := big.NewInt(0)

	for lvl := side.head; lvl != nil && remaining.Sign() > 0; lvl = lvl.next {
		if !side.priceCrosses(limitPrice, lvl.price, takerIsBuy) {
			break
		}
		for e := lvl.queue.Front(); e != nil && remaining.Sign() > 0; e = e.Next() {
			entry := e.Value.(*bookEntry)
			if entry == nil || entry.order == nil || entry.remain.Sign() <= 0 {
				continue
			}
			if skipSigner != "" && strings.EqualFold(entry.order.Signer, skipSigner) {
				continue
			}
			fill := minBig(remaining, entry.remain)
			if fill.Sign() <= 0 {
				continue
			}
			out.Levels = append(out.Levels, SimulatedFillLevel{
				PriceRaw:    lvl.price.String(),
				AmountPaper: fill.String(),
			})
			notional.Add(notional, new(big.Int).Mul(lvl.price, fill))
			out.FilledPaper.Add(out.FilledPaper, fill)
			remaining.Sub(remaining, fill)
			if worst.Sign() == 0 {
				worst = new(big.Int).Set(lvl.price)
			} else if takerIsBuy {
				if lvl.price.Cmp(worst) > 0 {
					worst.Set(lvl.price)
				}
			} else if lvl.price.Cmp(worst) < 0 {
				worst.Set(lvl.price)
			}
		}
	}

	out.UnfilledPaper = remaining
	if out.FilledPaper.Sign() > 0 {
		out.AvgPriceRaw = new(big.Int).Div(notional, out.FilledPaper)
		out.WorstPriceRaw = worst
	}
	return out
}
