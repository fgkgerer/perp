package account

import (
	"context"

	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListWithdrawsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 查询取款记录
func NewListWithdrawsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListWithdrawsLogic {
	return &ListWithdrawsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListWithdrawsLogic) ListWithdraws(req *types.ListWithdrawsReq) (resp *types.ListWithdrawsResp, err error) {
	// todo: add your logic here and delete this line

	return
}
