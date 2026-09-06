package order

import (
	"context"
	"strings"

	"metanode/internal/model"
	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CancelOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 取消订单
func NewCancelOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelOrderLogic {
	return &CancelOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CancelOrderLogic) CancelOrder(req *types.CancelOrderReq) (resp *types.CancelOrderResp, err error) {
	orderId := strings.TrimSpace(req.OrderId)
	signer := strings.TrimSpace(req.Signer)
	if orderId == "" || signer == "" {
		return &types.CancelOrderResp{Code: 400, Message: "orderId and signer are required"}, nil
	}

	o, err := l.svcCtx.OrderModel.FindOne(l.ctx, orderId)
	if err != nil || o == nil {
		return &types.CancelOrderResp{Code: 404, Message: "Order not found"}, nil
	}
	if !strings.EqualFold(o.Signer, signer) {
		return &types.CancelOrderResp{Code: 403, Message: "Signer mismatch"}, nil
	}
	if o.Status == model.OrderStatusFilled {
		return &types.CancelOrderResp{Code: 400, Message: "Order already filled"}, nil
	}
	if o.Status == model.OrderStatusCancelled {
		return &types.CancelOrderResp{Code: 0, Message: "success"}, nil
	}

	if l.svcCtx.MatchEngine != nil {
		if removed := l.svcCtx.MatchEngine.RemoveOrder(orderId); !removed {
			return &types.CancelOrderResp{Code: 409, Message: "Order is not cancelable in memory, it may already be matching"}, nil
		}
	}

	if err := l.svcCtx.OrderModel.UpdateStatus(l.ctx, orderId, model.OrderStatusCancelled, o.FilledAmount); err != nil {
		l.Error("Failed to cancel order:", err)
		return &types.CancelOrderResp{Code: 500, Message: "Failed to cancel order"}, nil
	}

	return &types.CancelOrderResp{Code: 0, Message: "success"}, nil
}
