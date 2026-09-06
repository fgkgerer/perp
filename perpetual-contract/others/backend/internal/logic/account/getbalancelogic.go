package account

// GetBalance：PrimaryCredit = Dealer.getCreditOf 链上主资产 credit + 链下 ledger_balances（USDC 转入 address_a 累计）。

import (
	"context"
	"math/big"

	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetBalanceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetBalanceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetBalanceLogic {
	return &GetBalanceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetBalanceLogic) GetBalance(req *types.GetBalanceReq) (resp *types.GetBalanceResp, err error) {
	if l.svcCtx.LedgerBalanceModel == nil {
		return &types.GetBalanceResp{Code: 503, Message: "database unavailable: configure Supabase.DataSource"}, nil
	}

	ledgerAmt, lerr := l.svcCtx.LedgerBalanceModel.GetPrimary(l.ctx, req.Trader)
	if lerr != nil {
		return &types.GetBalanceResp{Code: 500, Message: lerr.Error()}, nil
	}

	pc := big.NewInt(0)
	sc := big.NewInt(0)
	pp := big.NewInt(0)
	ps := big.NewInt(0)
	ts := big.NewInt(0)

	if l.svcCtx.Chain != nil {
		var cerr error
		pc, sc, pp, ps, ts, cerr = l.svcCtx.Chain.GetCreditOf(l.ctx, common.HexToAddress(req.Trader))
		if cerr != nil {
			// RPC/链上读失败时仍返回链下 ledger 余额，避免整接口 500
			l.Errorf("GetCreditOf %s: %v", req.Trader, cerr)
		}
	}

	primaryMerged := new(big.Int).Add(pc, ledgerAmt)
	return &types.GetBalanceResp{
		Code:    0,
		Message: "ok",
		Balance: types.AccountBalance{
			Trader:                   req.Trader,
			PrimaryCredit:            primaryMerged.String(),
			OnChainPrimaryCredit:     pc.String(),
			LedgerPrimaryBalance:     ledgerAmt.String(),
			SecondaryCredit:          sc.String(),
			PendingPrimaryWithdraw:   pp.String(),
			PendingSecondaryWithdraw: ps.String(),
			ExecutionTimestamp:       ts.Int64(),
		},
	}, nil
}
