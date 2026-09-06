package market

import (
	"metanode/internal/listener"
	"metanode/internal/svc"
	"metanode/internal/types"
)

// applyIndexPrice 将现货指数写入市场 DTO（前端展示 2 位小数）。
func applyIndexPrice(pm *types.PerpMarket, src svc.IndexPriceSource, perpAddress string) {
	if src == nil {
		return
	}
	if d, ok := src.IndexPriceDisplay(perpAddress); ok {
		pm.IndexPrice = d
		return
	}
	if raw, ok := src.IndexPrice1e6(perpAddress); ok {
		pm.IndexPrice = listener.FormatIndexDisplayFrom1e6(raw)
	}
}
