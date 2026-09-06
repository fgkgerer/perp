package market

// ChainTickers 等：行情数据直接读 Dealer 视图，与 src/MetaNodeView 一致。

import (
	"context"
	"math/big"
	"time"

	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/ethereum/go-ethereum/common"
)

// ChainTickers 从链上 Dealer 拉取各市场的标记价格与资金费率（用于 Ticker / WS）。
func ChainTickers(ctx context.Context, svc *svc.ServiceContext) []types.Ticker {
	if svc.Chain == nil {
		return nil
	}
	var out []types.Ticker
	for _, m := range svc.Config.Markets {
		perp := common.HexToAddress(m.Address)
		mp, err := svc.Chain.GetMarkPrice(ctx, perp)
		if err != nil {
			mp = big.NewInt(0)
		}
		fr, err := svc.Chain.GetFundingRate(ctx, perp)
		if err != nil {
			fr = big.NewInt(0)
		}
		out = append(out, types.Ticker{
			Perp:               m.Address,
			Price:              mp.String(),
			PriceChange:        "0",
			PriceChangePercent: "0",
			High24h:            mp.String(),
			Low24h:             mp.String(),
			Volume24h:          "0",
			UpdateTime:         time.Now().Unix(),
		})
		_ = fr // 如需下发可把 FundingRate 扩到 Ticker 类型
	}
	return out
}
