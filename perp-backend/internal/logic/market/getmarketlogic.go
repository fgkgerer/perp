package market

// GetMarket：按 address 在配置中查找，并附加链上 MarkPrice / FundingRate。

import (
	"context"

	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetMarketLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetMarketLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMarketLogic {
	return &GetMarketLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMarketLogic) GetMarket(req *types.GetMarketReq) (resp *types.GetMarketResp, err error) {
	var found types.PerpMarket
	for _, m := range l.svcCtx.Config.Markets {
		if m.Address == req.Address {
			found = types.PerpMarket{Address: m.Address, Name: m.Name, IsRegistered: true}
			if l.svcCtx.Chain != nil {
				perp := common.HexToAddress(m.Address)
				if mp, err := l.svcCtx.Chain.GetMarkPrice(l.ctx, perp); err == nil {
					found.MarkPrice = mp.String()
					found.IndexPrice = mp.String()
				}
				if fr, err := l.svcCtx.Chain.GetFundingRate(l.ctx, perp); err == nil {
					found.FundingRate = fr.String()
				}
			}
			return &types.GetMarketResp{Code: 0, Message: "ok", Market: found}, nil
		}
	}
	return &types.GetMarketResp{Code: 404, Message: "market not found"}, nil
}
