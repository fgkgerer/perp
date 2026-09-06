package market

import (
	"context"
	"strings"

	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetOpenQuoteLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetOpenQuoteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOpenQuoteLogic {
	return &GetOpenQuoteLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetOpenQuoteLogic) GetOpenQuote(req *types.GetOpenQuoteReq) (*types.GetOpenQuoteResp, error) {
	perp := strings.TrimSpace(req.Perp)
	side := strings.ToLower(strings.TrimSpace(req.Side))
	if perp == "" {
		return &types.GetOpenQuoteResp{Code: 400, Message: "perp is required"}, nil
	}
	if side != "long" && side != "short" {
		return &types.GetOpenQuoteResp{Code: 400, Message: "side must be long or short"}, nil
	}

	q := ResolveOpenQuote(l.ctx, l.svcCtx, perp, side)
	if q.PriceUsd == "" || q.PriceUsd == "0.00" {
		return &types.GetOpenQuoteResp{Code: 404, Message: "no quote available"}, nil
	}
	return &types.GetOpenQuoteResp{
		Code:    0,
		Message: "ok",
		Quote: types.OpenQuote{
			Perp:       perp,
			Side:       side,
			PriceUsd:   q.PriceUsd,
			PriceRaw:   q.PriceRaw,
			Source:     q.Source,
			BestBid:    q.BestBid,
			BestAsk:    q.BestAsk,
			MarkPrice:  q.MarkPrice,
			IndexPrice: q.IndexPrice,
		},
	}, nil
}
