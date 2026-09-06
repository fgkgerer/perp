package funding

import (
	"context"
	"strings"

	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListFundingRatesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取资金费率历史
func NewListFundingRatesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListFundingRatesLogic {
	return &ListFundingRatesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListFundingRatesLogic) ListFundingRates(req *types.ListFundingRatesReq) (resp *types.ListFundingRatesResp, err error) {
	perp := strings.TrimSpace(req.Perp)
	if perp == "" {
		return &types.ListFundingRatesResp{Code: 400, Message: "perp is required"}, nil
	}
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	if l.svcCtx.FundingRateModel == nil {
		return &types.ListFundingRatesResp{
			Code: 0, Message: "ok",
			FundingRates: []types.FundingRateRecord{},
			Page: page, PageSize: pageSize,
		}, nil
	}
	rows, total, err := l.svcCtx.FundingRateModel.FindByPerp(l.ctx, perp, page, pageSize)
	if err != nil {
		return nil, err
	}
	out := make([]types.FundingRateRecord, 0, len(rows))
	for _, r := range rows {
		out = append(out, types.FundingRateRecord{
			Perp:       r.Perp,
			Rate:       r.Rate,
			MarkPrice:  r.MarkPrice,
			IndexPrice: r.IndexPrice,
			SettleTime: r.SettleTime.Unix(),
		})
	}
	return &types.ListFundingRatesResp{
		Code:         0,
		Message:      "ok",
		FundingRates: out,
		Total:        total,
		Page:         page,
		PageSize:     pageSize,
	}, nil
}
