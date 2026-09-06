package market

// ChainTickers 等：行情数据直接读 Dealer 视图，与 src/MetaNodeView 一致。

import (
	"context"
	"time"

	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/ethereum/go-ethereum/common"
)

// ChainTickers 各市场 Ticker：Price 优先订单簿 mid，否则链上 mark（均为展示用 2 位小数）。
func ChainTickers(ctx context.Context, svc *svc.ServiceContext) []types.Ticker {
	var out []types.Ticker
	for _, m := range svc.Config.Markets {
		display := ResolveMarkPrice(ctx, svc, m.Address)
		if display == "" {
			display = "0.00"
		}
		out = append(out, types.Ticker{
			Perp:               m.Address,
			Price:              display,
			PriceChange:        "0",
			PriceChangePercent: "0",
			High24h:            display,
			Low24h:             display,
			Volume24h:          "0",
			UpdateTime:         time.Now().Unix(),
		})
		if svc.Chain != nil {
			_, _ = svc.Chain.GetFundingRate(ctx, common.HexToAddress(m.Address))
		}
	}
	return out
}
