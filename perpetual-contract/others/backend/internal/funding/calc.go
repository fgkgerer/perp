package funding

import (
	"fmt"
	"math/big"
	"strings"
)

// DeltaFromPremium 与 FundingRateKeeper.fundingDelta 一致：premium=(mark-index)/index，步长 premium/3。
func DeltaFromPremium(mark, index *big.Int, maxStep *big.Int) *big.Int {
	if index == nil || index.Sign() == 0 || mark == nil {
		return big.NewInt(0)
	}
	premium := new(big.Int).Sub(mark, index)
	premium.Mul(premium, big.NewInt(1e18))
	premium.Div(premium, index)
	step := new(big.Int).Div(premium, big.NewInt(3))
	if maxStep != nil && maxStep.Sign() > 0 {
		if step.Cmp(maxStep) > 0 {
			step.Set(maxStep)
		}
		negMax := new(big.Int).Neg(maxStep)
		if step.Cmp(negMax) < 0 {
			step.Set(negMax)
		}
	}
	return step
}

// PeriodRatePercent 将累计指数步长 delta（1e18 精度）格式化为当期费率百分比字符串，如 "0.0123"。
func PeriodRatePercent(delta *big.Int) string {
	if delta == nil || delta.Sign() == 0 {
		return "0.0000"
	}
	f := new(big.Float).SetInt(delta)
	f.Mul(f, big.NewFloat(100))
	f.Quo(f, big.NewFloat(1e18))
	s, _ := f.Float64()
	return fmt.Sprintf("%.4f", s)
}

// ParseMaxRateStep 解析配置 MaxRate（1e18 精度整数串）。
func ParseMaxRateStep(s string) *big.Int {
	z := new(big.Int)
	if _, ok := z.SetString(strings.TrimSpace(s), 10); !ok || z.Sign() <= 0 {
		return nil
	}
	return z
}

// DeltaBetweenCumulative 计算两次结算后链上累计指数之差（本期步长）。
func DeltaBetweenCumulative(newer, older string) *big.Int {
	a, okA := new(big.Int).SetString(strings.TrimSpace(newer), 10)
	b, okB := new(big.Int).SetString(strings.TrimSpace(older), 10)
	if !okA || !okB {
		return big.NewInt(0)
	}
	return new(big.Int).Sub(a, b)
}
