package market

import (
	"context"
	"fmt"
	"math/big"

	"metanode/internal/config"
	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/ethereum/go-ethereum/common"
)

const ratioOne = 1e18

func applyRiskParams(pm *types.PerpMarket, ctx context.Context, svcCtx *svc.ServiceContext, perpAddress string, mc config.MarketConfig) {
	applyRiskParamsFromConfig(pm, mc)
	if svcCtx.Chain == nil {
		return
	}
	rp, err := svcCtx.Chain.GetRiskParams(ctx, common.HexToAddress(perpAddress))
	if err != nil || rp == nil || !rp.IsRegistered {
		return
	}
	if rp.InitialMarginRatio != nil && rp.InitialMarginRatio.Sign() > 0 {
		pm.InitialMarginRatio = rp.InitialMarginRatio.String()
		pm.InitialMarginPct = ratioToPercentDisplay(rp.InitialMarginRatio)
		pm.MaxLeverage = maxLeverageFromIMR(rp.InitialMarginRatio)
	}
	if rp.LiquidationThreshold != nil && rp.LiquidationThreshold.Sign() > 0 {
		pm.LiquidationThreshold = rp.LiquidationThreshold.String()
		pm.MaintenanceMarginPct = ratioToPercentDisplay(rp.LiquidationThreshold)
	}
	if rp.LiquidationPriceOff != nil {
		pm.LiquidationPriceOff = rp.LiquidationPriceOff.String()
	}
	if rp.InsuranceFeeRate != nil {
		pm.InsuranceFeeRate = rp.InsuranceFeeRate.String()
	}
}

func applyRiskParamsFromConfig(pm *types.PerpMarket, mc config.MarketConfig) {
	if pm.MaxLeverage == "" && mc.MaxLeverage != "" {
		pm.MaxLeverage = mc.MaxLeverage
	}
	if pm.InitialMarginPct == "" && mc.InitialMarginPct != "" {
		pm.InitialMarginPct = mc.InitialMarginPct
	}
	if pm.MaintenanceMarginPct == "" && mc.MaintenanceMarginPct != "" {
		pm.MaintenanceMarginPct = mc.MaintenanceMarginPct
	}
}

func ratioToPercentDisplay(r *big.Int) string {
	if r == nil || r.Sign() <= 0 {
		return ""
	}
	f := new(big.Float).SetInt(r)
	f.Mul(f, big.NewFloat(100))
	f.Quo(f, big.NewFloat(ratioOne))
	v, _ := f.Float64()
	return fmt.Sprintf("%.2f", v)
}

// maxLeverageFromIMR 最大杠杆 ≈ 1 / 初始保证金率（1e18 精度整数除）。
func maxLeverageFromIMR(imr *big.Int) string {
	if imr == nil || imr.Sign() <= 0 {
		return ""
	}
	one := big.NewInt(ratioOne)
	if new(big.Int).Mod(one, imr).Sign() != 0 {
		f := new(big.Float).SetInt(one)
		f.Quo(f, new(big.Float).SetInt(imr))
		v, _ := f.Float64()
		return fmt.Sprintf("%.1f", v)
	}
	lev := new(big.Int).Div(one, imr)
	return lev.String()
}
