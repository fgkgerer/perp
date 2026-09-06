package order

import (
	"context"

	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOrderLogic {
	return &GetOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetOrder 查询订单
func (l *GetOrderLogic) GetOrder(req *types.GetOrderReq) (resp *types.GetOrderResp, err error) {
	order, err := l.svcCtx.OrderModel.FindOne(l.ctx, req.OrderId)
	if err != nil {
		return &types.GetOrderResp{Code: 404, Message: "Order not found"}, nil
	}

	return &types.GetOrderResp{
		Code:    0,
		Message: "success",
		Order: types.Order{
			OrderId:      order.OrderId,
			Perp:         order.Perp,
			Signer:       order.Signer,
			PaperAmount:  order.PaperAmount,
			CreditAmount: order.CreditAmount,
			MakerFeeRate: order.MakerFeeRate,
			TakerFeeRate: order.TakerFeeRate,
			Expiration:   order.Expiration,
			Nonce:        order.Nonce,
			Signature:    order.Signature,
			Status:       order.Status,
			FilledAmount: order.FilledAmount,
			CreateTime:   order.CreateTime.Unix(),
			UpdateTime:   order.UpdateTime.Unix(),
		},
	}, nil
}
