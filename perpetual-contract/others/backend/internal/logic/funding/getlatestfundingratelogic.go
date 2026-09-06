package funding

import (
	"context"
	"strings"
	"time"

	fundingcalc "metanode/internal/funding"
	"metanode/internal/logic/market"
	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetLatestFundingRateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetLatestFundingRateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetLatestFundingRateLogic {
	return &GetLatestFundingRateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetLatestFundingRateLogic) GetLatestFundingRate(req *types.GetLatestFundingRateReq) (*types.GetLatestFundingRateResp, error) {
	perp := strings.TrimSpace(req.Perp)
	if perp == "" {
		return &types.GetLatestFundingRateResp{Code: 400, Message: "perp is required"}, nil
	}

	intervalSec := l.svcCtx.Config.FundingRate.SettleInterval
	if intervalSec <= 0 {
		intervalSec = 28800
	}
	now := time.Now()
	maxStep := fundingcalc.ParseMaxRateStep(l.svcCtx.Config.FundingRate.MaxRate)

	out := types.FundingRateLatest{
		Perp:              perp,
		NextSettleAt:      fundingcalc.NextSettleUnix(now, intervalSec),
		SettleIntervalSec: intervalSec,
		SettleSchedule:    fundingcalc.ScheduleLabel(intervalSec),
		PeriodRateSource:  "predicted",
		PeriodRatePct:     "0.0000",
	}

	mark, idx := market.MarkIndex1e6ForFunding(l.ctx, l.svcCtx, perp)
	predicted := fundingcalc.DeltaFromPremium(mark, idx, maxStep)
	out.PredictedPeriodRatePct = fundingcalc.PeriodRatePercent(predicted)

	if l.svcCtx.Chain != nil {
		if fr, err := l.svcCtx.Chain.GetFundingRate(l.ctx, common.HexToAddress(perp)); err == nil {
			out.CumulativeRate = fr.String()
		}
	}

	if l.svcCtx.FundingRateModel != nil {
		records, _, err := l.svcCtx.FundingRateModel.FindByPerp(l.ctx, perp, 1, 2)
		if err == nil && len(records) > 0 {
			out.LastSettleAt = records[0].SettleTime.Unix()
			if len(records) >= 2 {
				delta := fundingcalc.DeltaBetweenCumulative(records[0].Rate, records[1].Rate)
				out.PeriodRatePct = fundingcalc.PeriodRatePercent(delta)
				out.PeriodRateSource = "last_settle"
			} else {
				out.PeriodRatePct = out.PredictedPeriodRatePct
			}
		} else {
			out.PeriodRatePct = out.PredictedPeriodRatePct
		}
	} else {
		out.PeriodRatePct = out.PredictedPeriodRatePct
	}

	return &types.GetLatestFundingRateResp{
		Code:    0,
		Message: "ok",
		Latest:  out,
	}, nil
}
