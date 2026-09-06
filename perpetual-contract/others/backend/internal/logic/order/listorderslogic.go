package order

import (
	"context"

	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListOrdersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListOrdersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListOrdersLogic {
	return &ListOrdersLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ListOrders 查询订单列表
func (l *ListOrdersLogic) ListOrders(req *types.ListOrdersReq) (resp *types.ListOrdersResp, err error) {
	orders, total, err := l.svcCtx.OrderModel.FindByTrader(l.ctx, req.Signer, req.Perp, req.Status, req.Page, req.PageSize)
	if err != nil {
		l.Error("Failed to list orders:", err)
		return &types.ListOrdersResp{Code: 500, Message: "Failed to list orders"}, nil
	}

	var orderList []types.Order
	for _, order := range orders {
		orderList = append(orderList, types.Order{
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
		})
	}

	return &types.ListOrdersResp{
		Code:     0,
		Message:  "success",
		Orders:   orderList,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}
