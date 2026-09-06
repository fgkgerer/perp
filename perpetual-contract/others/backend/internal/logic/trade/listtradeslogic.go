package trade

import (
	"context"
	"strings"

	"metanode/internal/model"
	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListTradesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 查询成交记录
func NewListTradesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListTradesLogic {
	return &ListTradesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListTradesLogic) ListTrades(req *types.ListTradesReq) (resp *types.ListTradesResp, err error) {
	if l.svcCtx.TradeModel == nil {
		return &types.ListTradesResp{
			Code:     0,
			Message:  "ok",
			Trades:   []types.Trade{},
			Total:    0,
			Page:     1,
			PageSize: req.PageSize,
		}, nil
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	trader := strings.TrimSpace(req.Trader)
	perp := strings.TrimSpace(req.Perp)

	if trader != "" {
		rows, total, err := l.svcCtx.TradeModel.FindByTrader(l.ctx, trader, perp, page, pageSize)
		if err != nil {
			return nil, err
		}
		return &types.ListTradesResp{
			Code:     0,
			Message:  "ok",
			Trades:   toTradeDTOs(rows),
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}

	if perp == "" {
		return &types.ListTradesResp{Code: 400, Message: "perp or trader is required"}, nil
	}

	limit := pageSize
	if page > 1 {
		limit = page * pageSize
	}
	rows, err := l.svcCtx.TradeModel.FindRecent(l.ctx, perp, limit)
	if err != nil {
		return nil, err
	}
	total := int64(len(rows))
	if page > 1 && len(rows) > pageSize {
		start := (page - 1) * pageSize
		if start >= len(rows) {
			rows = nil
		} else {
			end := start + pageSize
			if end > len(rows) {
				end = len(rows)
			}
			rows = rows[start:end]
		}
	}
	return &types.ListTradesResp{
		Code:     0,
		Message:  "ok",
		Trades:   toTradeDTOs(rows),
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func toTradeDTOs(rows []*model.Trade) []types.Trade {
	if len(rows) == 0 {
		return []types.Trade{}
	}
	out := make([]types.Trade, 0, len(rows))
	for _, r := range rows {
		if r == nil {
			continue
		}
		out = append(out, types.Trade{
			TradeId:      r.TradeId,
			Perp:         r.Perp,
			TakerOrderId: r.TakerOrderId,
			MakerOrderId: r.MakerOrderId,
			Taker:        r.Taker,
			Maker:        r.Maker,
			PaperAmount:  r.PaperAmount,
			Price:        r.Price,
			TakerFee:     r.TakerFee,
			MakerFee:     r.MakerFee,
			TxHash:       r.TxHash,
			BlockNumber:  r.BlockNumber,
			CreateTime:   r.CreateTime.Unix(),
		})
	}
	return out
}
