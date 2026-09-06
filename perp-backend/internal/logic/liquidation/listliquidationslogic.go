package liquidation

import (
	"context"

	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListLiquidationsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取清算记录
func NewListLiquidationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListLiquidationsLogic {
	return &ListLiquidationsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListLiquidationsLogic) ListLiquidations(req *types.ListLiquidationsReq) (resp *types.ListLiquidationsResp, err error) {
	// todo: add your logic here and delete this line

	return
}
