package position

// GetRiskInfo：净值/敞口/维持保证金取 Dealer.getTraderRisk；是否安全取 isSafe。

import (
	"context"
	"math/big"

	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetRiskInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetRiskInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRiskInfoLogic {
	return &GetRiskInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetRiskInfoLogic) GetRiskInfo(req *types.GetRiskInfoReq) (resp *types.GetRiskInfoResp, err error) {
	if l.svcCtx.Chain == nil {
		return &types.GetRiskInfoResp{Code: 0, Message: "ok"}, nil
	}
	trader := common.HexToAddress(req.Trader)
	nv, exp, _, maint, err := l.svcCtx.Chain.GetTraderRisk(l.ctx, trader)
	if err != nil {
		return &types.GetRiskInfoResp{Code: 500, Message: err.Error()}, nil
	}
	safe, _ := l.svcCtx.Chain.IsSafe(l.ctx, trader)
	avail := new(big.Int).Sub(nv, maint)
	ratio := "0"
	if exp.Sign() > 0 && maint.Sign() > 0 {
		ratio = new(big.Int).Div(new(big.Int).Mul(nv, big.NewInt(1e18)), exp).String()
	}
	return &types.GetRiskInfoResp{
		Code:    0,
		Message: "ok",
		RiskInfo: types.RiskInfo{
			Trader:            req.Trader,
			NetValue:          nv.String(),
			Exposure:          exp.String(),
			MaintenanceMargin: maint.String(),
			AvailableMargin:   avail.String(),
			MarginRatio:       ratio,
			IsSafe:            safe,
		},
	}, nil
}
