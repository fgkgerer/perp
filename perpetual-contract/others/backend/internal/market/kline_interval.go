package market

import (
	"fmt"
	"strings"
	"time"
)

// NormalizeKlineInterval 统一周期：支持 15m/1h/4h/1d 及前端 TradingView 风格 15/60/240/1D。
func NormalizeKlineInterval(raw string) (string, error) {
	s := strings.TrimSpace(strings.ToLower(raw))
	switch s {
	case "1m":
		return "1m", nil
	case "5m":
		return "5m", nil
	case "15m", "15":
		return "15m", nil
	case "1h", "60":
		return "1h", nil
	case "4h", "240":
		return "4h", nil
	case "1d", "1day", "d":
		return "1d", nil
	default:
		if strings.HasSuffix(s, "m") || strings.HasSuffix(s, "h") || strings.HasSuffix(s, "d") {
			return s, nil
		}
		return "", fmt.Errorf("unsupported interval: %s", raw)
	}
}

// IntervalDuration 返回 K 线周期时长。
func IntervalDuration(interval string) (time.Duration, error) {
	iv, err := NormalizeKlineInterval(interval)
	if err != nil {
		return 0, err
	}
	switch iv {
	case "1m":
		return time.Minute, nil
	case "5m":
		return 5 * time.Minute, nil
	case "15m":
		return 15 * time.Minute, nil
	case "1h":
		return time.Hour, nil
	case "4h":
		return 4 * time.Hour, nil
	case "1d":
		return 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported interval: %s", interval)
	}
}

// AlignOpenTime 将时间戳对齐到 K 线开盘时刻（UTC）。
func AlignOpenTime(t time.Time, interval string) (time.Time, error) {
	dur, err := IntervalDuration(interval)
	if err != nil {
		return time.Time{}, err
	}
	sec := int64(dur.Seconds())
	if sec <= 0 {
		return time.Time{}, fmt.Errorf("invalid interval duration")
	}
	u := t.UTC().Unix()
	aligned := (u / sec) * sec
	return time.Unix(aligned, 0).UTC(), nil
}

// BinanceInterval 映射到 Binance REST klines interval 参数。
func BinanceInterval(interval string) (string, error) {
	iv, err := NormalizeKlineInterval(interval)
	if err != nil {
		return "", err
	}
	switch iv {
	case "1m", "5m", "15m", "1h", "4h", "1d":
		return iv, nil
	default:
		return "", fmt.Errorf("binance unsupported interval: %s", interval)
	}
}

// AllRecordingIntervals 现货指数落库使用的周期集合。
func AllRecordingIntervals() []string {
	return []string{"1m", "5m", "15m", "1h", "4h", "1d"}
}
