package market

// ListMarkets：市场静态信息来自配置，标记价/资金指数来自链上（若 Chain 可用）。

import (
	"context"

	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListMarketsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListMarketsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListMarketsLogic {
	return &ListMarketsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListMarketsLogic) ListMarkets(req *types.ListMarketsReq) (resp *types.ListMarketsResp, err error) {
	var markets []types.PerpMarket
	for _, m := range l.svcCtx.Config.Markets {
		pm := types.PerpMarket{
			Address:      m.Address,
			Name:         m.Name,
			IsRegistered: true,
		}
		if l.svcCtx.Chain != nil {
			perp := common.HexToAddress(m.Address)
			if mp, err := l.svcCtx.Chain.GetMarkPrice(l.ctx, perp); err == nil {
				pm.MarkPrice = mp.String()
				pm.IndexPrice = mp.String()
			}
			if fr, err := l.svcCtx.Chain.GetFundingRate(l.ctx, perp); err == nil {
				pm.FundingRate = fr.String()
			}
		}
		if l.svcCtx.IndexPrice != nil {
			if idx, ok := l.svcCtx.IndexPrice.IndexPrice1e6(m.Address); ok {
				pm.IndexPrice = idx
			}
		}
		markets = append(markets, pm)
	}
	return &types.ListMarketsResp{Code: 0, Message: "ok", Markets: markets}, nil
}
