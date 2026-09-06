package funding

import (
	"context"

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
	// todo: add your logic here and delete this line

	return
}
