package market

import (
	"math/big"
	"strings"
)

const defaultSlippageBps = 50 // 0.5%

// LimitPriceWithSlippage 由参考价与滑点计算签名限价（1e18 口径，与订单簿 price 一致）。
// 做多：限价 = 参考价 × (1 + bps/10000)，向上取整；做空：向下取整。
func LimitPriceWithSlippage(referenceRaw *big.Int, takerIsBuy bool, slippageBps int) *big.Int {
	if referenceRaw == nil || referenceRaw.Sign() <= 0 {
		return big.NewInt(0)
	}
	if slippageBps < 0 {
		slippageBps = 0
	}
	if slippageBps > 5000 {
		slippageBps = 5000
	}
	num := big.NewInt(int64(10000))
	if takerIsBuy {
		num.Add(num, big.NewInt(int64(slippageBps)))
	} else {
		num.Sub(num, big.NewInt(int64(slippageBps)))
		if num.Sign() <= 0 {
			return big.NewInt(1)
		}
	}
	out := new(big.Int).Mul(referenceRaw, num)
	out.Div(out, big.NewInt(10000))
	if takerIsBuy && out.Cmp(referenceRaw) < 0 {
		return new(big.Int).Set(referenceRaw)
	}
	if !takerIsBuy && out.Cmp(referenceRaw) > 0 {
		return new(big.Int).Set(referenceRaw)
	}
	return out
}

func NormalizeSlippageBps(bps int) int {
	if bps <= 0 {
		return defaultSlippageBps
	}
	if bps > 5000 {
		return 5000
	}
	return bps
}

func sideIsLong(side string) bool {
	return strings.EqualFold(strings.TrimSpace(side), "long")
}

// ParsePaperHuman 将人类可读数量（如 0.01）转为 paper 1e18 整数。
func ParsePaperHuman(sizeHuman string) (*big.Int, error) {
	s := strings.TrimSpace(sizeHuman)
	if s == "" {
		return nil, errInvalidSize
	}
	f, ok := new(big.Float).SetString(s)
	if !ok || f.Sign() <= 0 {
		return nil, errInvalidSize
	}
	scale := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	f.Mul(f, scale)
	out, _ := f.Int(nil)
	if out == nil || out.Sign() <= 0 {
		return nil, errInvalidSize
	}
	return out, nil
}

// FormatPaperHuman paper 1e18 → 人类可读（去尾零）。
func FormatPaperHuman(paper *big.Int) string {
	if paper == nil || paper.Sign() <= 0 {
		return "0"
	}
	scale := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	f := new(big.Float).Quo(new(big.Float).SetInt(paper), scale)
	return strings.TrimRight(strings.TrimRight(f.Text('f', 8), "0"), ".")
}

var errInvalidSize = &parseError{"invalid size"}

type parseError struct{ msg string }

func (e *parseError) Error() string { return e.msg }
