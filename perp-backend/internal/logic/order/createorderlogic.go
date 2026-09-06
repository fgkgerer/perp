package order

// CreateOrder 与链上 Types.Order + EIP-712 对齐；orderId 即合约内用于累计成交量的 orderHash。

import (
	"context"
	"math/big"
	"time"

	"metanode/internal/chain"
	"metanode/internal/model"
	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateOrderLogic {
	return &CreateOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// CreateOrder 创建订单：校验 EIP-712 签名，写入 DB 并进入撮合引擎。
// 开仓与买卖使用挂单成交；平仓使用与持仓相反方向的订单，与对手盘撮合后由链上结算。
func (l *CreateOrderLogic) CreateOrder(req *types.CreateOrderReq) (resp *types.CreateOrderResp, err error) {
	if req.Expiration <= time.Now().Unix() {
		return &types.CreateOrderResp{Code: 400, Message: "Order expired"}, nil
	}

	order := &model.Order{
		ChainId:      l.svcCtx.Config.Ethereum.ChainId,
		Perp:         req.Perp,
		Signer:       req.Signer,
		PaperAmount:  req.PaperAmount,
		CreditAmount: req.CreditAmount,
		MakerFeeRate: req.MakerFeeRate,
		TakerFeeRate: req.TakerFeeRate,
		Expiration:   req.Expiration,
		Nonce:        req.Nonce,
		Signature:    req.Signature,
		Status:       model.OrderStatusPending,
		FilledAmount: "0",
		CreateTime:   time.Now(),
		UpdateTime:   time.Now(),
	}

	dc := chain.DealerChain{
		Dealer:  common.HexToAddress(l.svcCtx.Config.Ethereum.DealerAddress),
		ChainID: big.NewInt(l.svcCtx.Config.Ethereum.ChainId),
	}
	h, err := chain.OrderDigestWith(dc, order)
	if err != nil {
		return &types.CreateOrderResp{Code: 400, Message: "Invalid order fields: " + err.Error()}, nil
	}
	orderId := h.Hex()
	order.OrderId = orderId

	if l.svcCtx.Chain != nil {
		if err := chain.VerifyEOASignature(dc, order); err != nil {
			return &types.CreateOrderResp{Code: 400, Message: "Invalid signature: " + err.Error()}, nil
		}
	}

	existOrder, _ := l.svcCtx.OrderModel.FindOne(l.ctx, orderId)
	if existOrder != nil {
		return &types.CreateOrderResp{Code: 400, Message: "Order already exists"}, nil
	}

	if _, err := l.svcCtx.OrderModel.Insert(l.ctx, order); err != nil {
		l.Error("Failed to insert order:", err)
		return &types.CreateOrderResp{Code: 500, Message: "Failed to create order"}, nil
	}

	if l.svcCtx.MatchEngine != nil {
		l.svcCtx.MatchEngine.AddOrder(order)
	}

	return &types.CreateOrderResp{
		Code:    0,
		Message: "success",
		OrderId: orderId,
	}, nil
}
