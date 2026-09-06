package market

import (
	"context"
	"strings"
	"time"

	"metanode/internal/config"
	"metanode/internal/model"
	"metanode/internal/svc"
)

// BinanceSymbolForPerp 从配置读取 Binance 现货交易对。
func BinanceSymbolForPerp(cfg config.Config, perp string) string {
	perp = strings.ToLower(strings.TrimSpace(perp))
	for _, m := range cfg.Markets {
		if strings.ToLower(strings.TrimSpace(m.Address)) == perp {
			return strings.TrimSpace(m.BinanceSymbol)
		}
	}
	return ""
}

// RecordSpotIndexPrice 将现货指数价写入各周期 K 线（在 CompositeSpotIndex 节流保存时调用）。
func RecordSpotIndexPrice(ctx context.Context, svcCtx *svc.ServiceContext, perp, priceUsd string) {
	if svcCtx == nil || svcCtx.SpotKlineModel == nil {
		return
	}
	priceUsd = strings.TrimSpace(priceUsd)
	if priceUsd == "" {
		return
	}
	perp = strings.ToLower(strings.TrimSpace(perp))
	now := time.Now().UTC()
	for _, iv := range AllRecordingIntervals() {
		openTime, err := AlignOpenTime(now, iv)
		if err != nil {
			continue
		}
		bar := &model.SpotKline{
			Perp:         perp,
			IntervalType: iv,
			OpenTime:     openTime,
			OpenPrice:    priceUsd,
			HighPrice:    priceUsd,
			LowPrice:     priceUsd,
			ClosePrice:   priceUsd,
			Volume:       "0",
			UpdatedAt:    now,
		}
		_ = svcCtx.SpotKlineModel.UpsertBar(ctx, bar)
	}
}

// EnsureHistoricalKlines 若库内 K 线不足则从 Binance 回填。
func EnsureHistoricalKlines(
	ctx context.Context,
	svcCtx *svc.ServiceContext,
	perp, interval string,
	start, end time.Time,
	want int,
) error {
	if svcCtx == nil || svcCtx.SpotKlineModel == nil || svcCtx.PG == nil {
		return nil
	}
	symbol := BinanceSymbolForPerp(svcCtx.Config, perp)
	if symbol == "" {
		return nil
	}
	cnt, err := svcCtx.SpotKlineModel.CountBars(ctx, perp, interval, start, end)
	if err != nil {
		return err
	}
	if int(cnt) >= want/2 {
		return nil
	}

	iv, err := NormalizeKlineInterval(interval)
	if err != nil {
		return err
	}
	bars, err := FetchBinanceKlines(ctx, symbol, iv, start.UnixMilli(), end.UnixMilli(), want)
	if err != nil {
		return err
	}
	for i := range bars {
		bars[i].Perp = strings.ToLower(strings.TrimSpace(perp))
	}
	return model.BulkUpsertBars(ctx, svcCtx.PG, bars)
}
