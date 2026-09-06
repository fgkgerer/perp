package position

// GetPositions / GetRiskInfo：仓位与风险来自 Perp.balanceOf 与 Dealer 聚合查询。

import (
	"context"
	"math/big"

	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetPositionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetPositionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPositionsLogic {
	return &GetPositionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetPositionsLogic) GetPositions(req *types.GetPositionsReq) (resp *types.GetPositionsResp, err error) {
	if l.svcCtx.Chain == nil {
		return &types.GetPositionsResp{Code: 0, Message: "ok", Positions: nil}, nil
	}
	trader := common.HexToAddress(req.Trader)
	perps, err := l.svcCtx.Chain.GetPositions(l.ctx, trader)
	if err != nil {
		return &types.GetPositionsResp{Code: 500, Message: err.Error()}, nil
	}
	var out []types.Position
	for _, perp := range perps {
		paper, credit, err := l.svcCtx.Chain.BalanceOf(l.ctx, perp, trader)
		if err != nil || paper.Sign() == 0 {
			continue
		}
		mark, _ := l.svcCtx.Chain.GetMarkPrice(l.ctx, perp)
		liq, _ := l.svcCtx.Chain.GetLiquidationPrice(l.ctx, trader, perp)

		absP := new(big.Int).Abs(new(big.Int).Set(paper))
		ac := new(big.Int).Abs(credit)
		// 展示用近似开仓价：|credit|/|paper|（1e18 精度），与合约内 credit/paper 比一致。
		entryPrice := new(big.Int).Div(new(big.Int).Mul(ac, big.NewInt(1e18)), absP).String()

		name := ""
		for _, m := range l.svcCtx.Config.Markets {
			if common.HexToAddress(m.Address) == perp {
				name = m.Name
				break
			}
		}
		out = append(out, types.Position{
			Trader:     req.Trader,
			Perp:       perp.Hex(),
			PerpName:   name,
			Paper:      paper.String(),
			Credit:     credit.String(),
			EntryPrice: entryPrice,
			MarkPrice:  mark.String(),
			LiqPrice:   liq.String(),
		})
	}
	return &types.GetPositionsResp{Code: 0, Message: "ok", Positions: out}, nil
}
