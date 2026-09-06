package account

import (
	"context"
	"strings"

	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListDepositsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 查询存款记录
func NewListDepositsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListDepositsLogic {
	return &ListDepositsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListDepositsLogic) ListDeposits(req *types.ListDepositsReq) (resp *types.ListDepositsResp, err error) {
	trader := strings.TrimSpace(req.Trader)
	if trader == "" {
		return &types.ListDepositsResp{Code: 400, Message: "trader required", Page: req.Page, PageSize: req.PageSize}, nil
	}
	trader = strings.ToLower(trader)

	if l.svcCtx.DepositModel == nil {
		return &types.ListDepositsResp{
			Code: 503, Message: "database unavailable: configure Supabase.DataSource",
			Page: req.Page, PageSize: req.PageSize,
		}, nil
	}

	list, total, err := l.svcCtx.DepositModel.FindByTrader(l.ctx, trader, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}
	out := make([]types.DepositRecord, 0, len(list))
	for _, d := range list {
		out = append(out, types.DepositRecord{
			TxHash:          d.TxHash,
			Trader:          d.Trader,
			PrimaryAmount:   d.PrimaryAmount,
			SecondaryAmount: d.SecondaryAmount,
			BlockNumber:     d.BlockNumber,
			CreateTime:      d.CreateTime.Unix(),
		})
	}
	return &types.ListDepositsResp{
		Code:     0,
		Message:  "ok",
		Deposits: out,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}
