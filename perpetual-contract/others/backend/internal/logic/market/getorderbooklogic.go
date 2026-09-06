package market

import (
	"context"
	"strings"

	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetOrderBookLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取深度
func NewGetOrderBookLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOrderBookLogic {
	return &GetOrderBookLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetOrderBookLogic) GetOrderBook(req *types.GetOrderBookReq) (resp *types.GetOrderBookResp, err error) {
	perp := strings.TrimSpace(req.Perp)
	if perp == "" {
		return &types.GetOrderBookResp{Code: 400, Message: "perp is required"}, nil
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	var bids, asks []types.OrderBookEntry
	if l.svcCtx.OrderBook != nil {
		rawBids, rawAsks := l.svcCtx.OrderBook.SnapshotOrderBook(perp, limit)
		bids = toOrderBookEntries(rawBids)
		asks = toOrderBookEntries(rawAsks)
	}
	if bids == nil {
		bids = []types.OrderBookEntry{}
	}
	if asks == nil {
		asks = []types.OrderBookEntry{}
	}
	return &types.GetOrderBookResp{
		Code:    0,
		Message: "ok",
		Bids:    bids,
		Asks:    asks,
	}, nil
}

func toOrderBookEntries(levels []svc.OrderBookLevel) []types.OrderBookEntry {
	out := make([]types.OrderBookEntry, 0, len(levels))
	for _, lv := range levels {
		out = append(out, types.OrderBookEntry{
			Price:  lv.Price,
			Amount: lv.Amount,
		})
	}
	return out
}
