package market

import (
	"context"

	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetKlinesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取K线数据
func NewGetKlinesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetKlinesLogic {
	return &GetKlinesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetKlinesLogic) GetKlines(req *types.GetKlinesReq) (resp *types.GetKlinesResp, err error) {
	// todo: add your logic here and delete this line

	return
}
