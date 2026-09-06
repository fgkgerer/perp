package market

// GetTicker：REST 轮询链上 MarkPrice（与 ChainTickers / WS 数据源一致）。

import (
	"context"

	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTickerLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetTickerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTickerLogic {
	return &GetTickerLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetTickerLogic) GetTicker(req *types.GetTickerReq) (resp *types.GetTickerResp, err error) {
	tickers := ChainTickers(l.ctx, l.svcCtx)
	if req.Perp != "" {
		var filtered []types.Ticker
		for _, t := range tickers {
			if t.Perp == req.Perp {
				filtered = append(filtered, t)
				break
			}
		}
		tickers = filtered
	}
	return &types.GetTickerResp{Code: 0, Message: "ok", Tickers: tickers}, nil
}
