package market

import (
	"context"
	"strings"
	"time"

	marketpkg "metanode/internal/market"
	"metanode/internal/model"
	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetKlinesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetKlinesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetKlinesLogic {
	return &GetKlinesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetKlinesLogic) GetKlines(req *types.GetKlinesReq) (*types.GetKlinesResp, error) {
	perp := strings.ToLower(strings.TrimSpace(req.Perp))
	if perp == "" {
		return &types.GetKlinesResp{Code: 400, Message: "perp is required"}, nil
	}

	intervalRaw := strings.TrimSpace(req.Interval)
	if intervalRaw == "" {
		intervalRaw = "15m"
	}
	interval, err := marketpkg.NormalizeKlineInterval(intervalRaw)
	if err != nil {
		return &types.GetKlinesResp{Code: 400, Message: err.Error()}, nil
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	dur, err := marketpkg.IntervalDuration(interval)
	if err != nil {
		return &types.GetKlinesResp{Code: 400, Message: err.Error()}, nil
	}

	now := time.Now().UTC()
	end := now
	if req.EndTime > 0 {
		end = time.Unix(req.EndTime, 0).UTC()
	}
	start := end.Add(-time.Duration(limit) * dur)
	if req.StartTime > 0 {
		start = time.Unix(req.StartTime, 0).UTC()
	}
	if !start.Before(end) {
		start = end.Add(-time.Duration(limit) * dur)
	}

	if l.svcCtx.SpotKlineModel != nil {
		if err := marketpkg.EnsureHistoricalKlines(l.ctx, l.svcCtx, perp, interval, start, end, limit); err != nil {
			l.Infof("kline backfill %s %s: %v", perp, interval, err)
		}
	}

	var rows []model.SpotKline
	if l.svcCtx.SpotKlineModel != nil {
		rows, err = l.svcCtx.SpotKlineModel.FindBars(l.ctx, perp, interval, start, end, limit)
		if err != nil {
			return &types.GetKlinesResp{Code: 500, Message: err.Error()}, nil
		}
	}

	klines := make([]types.Kline, 0, len(rows))
	for _, row := range rows {
		klines = append(klines, types.Kline{
			Time:   row.OpenTime.Unix(),
			Open:   row.OpenPrice,
			High:   row.HighPrice,
			Low:    row.LowPrice,
			Close:  row.ClosePrice,
			Volume: row.Volume,
		})
	}

	// 库中仍无数据时，尝试用内存中的现货指数拼一根当前 K 线，避免图表全空
	if len(klines) == 0 && l.svcCtx.IndexPrice != nil {
		if px, ok := l.svcCtx.IndexPrice.IndexPriceDisplay(perp); ok && strings.TrimSpace(px) != "" {
			openTime, aerr := marketpkg.AlignOpenTime(now, interval)
			if aerr == nil {
				klines = append(klines, types.Kline{
					Time:   openTime.Unix(),
					Open:   px,
					High:   px,
					Low:    px,
					Close:  px,
					Volume: "0",
				})
			}
		}
	}

	if len(klines) == 0 {
		return &types.GetKlinesResp{
			Code:    0,
			Message: "ok",
			Klines:  []types.Kline{},
		}, nil
	}

	return &types.GetKlinesResp{
		Code:    0,
		Message: "ok",
		Klines:  klines,
	}, nil
}
