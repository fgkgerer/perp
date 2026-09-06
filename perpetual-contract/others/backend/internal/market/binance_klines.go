package market

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"metanode/internal/model"
)

const binanceKlinesURL = "https://api.binance.com/api/v3/klines"

// FetchBinanceKlines 从 Binance 现货拉取历史 K 线（用于首次回填）。
func FetchBinanceKlines(
	ctx context.Context,
	symbol string,
	interval string,
	startMs, endMs int64,
	limit int,
) ([]model.SpotKline, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return nil, fmt.Errorf("empty binance symbol")
	}
	bi, err := BinanceInterval(interval)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 500
	}
	if limit > 1000 {
		limit = 1000
	}

	q := fmt.Sprintf("%s?symbol=%s&interval=%s&limit=%d", binanceKlinesURL, symbol, bi, limit)
	if startMs > 0 {
		q += "&startTime=" + strconv.FormatInt(startMs, 10)
	}
	if endMs > 0 {
		q += "&endTime=" + strconv.FormatInt(endMs, 10)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, q, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("binance klines HTTP %d: %s", res.StatusCode, string(body))
	}

	var raw [][]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	iv, _ := NormalizeKlineInterval(interval)
	now := time.Now().UTC()
	out := make([]model.SpotKline, 0, len(raw))
	for _, row := range raw {
		if len(row) < 6 {
			continue
		}
		openMs, err := rawInt64(row[0])
		if err != nil {
			continue
		}
		openP, _ := rawString(row[1])
		highP, _ := rawString(row[2])
		lowP, _ := rawString(row[3])
		closeP, _ := rawString(row[4])
		vol, _ := rawString(row[5])
		out = append(out, model.SpotKline{
			IntervalType: iv,
			OpenTime:     time.UnixMilli(openMs).UTC(),
			OpenPrice:    openP,
			HighPrice:    highP,
			LowPrice:     lowP,
			ClosePrice:   closeP,
			Volume:       vol,
			UpdatedAt:    now,
		})
	}
	return out, nil
}

func rawInt64(r json.RawMessage) (int64, error) {
	var n int64
	if err := json.Unmarshal(r, &n); err == nil {
		return n, nil
	}
	var s string
	if err := json.Unmarshal(r, &s); err != nil {
		return 0, err
	}
	return strconv.ParseInt(s, 10, 64)
}

func rawString(r json.RawMessage) (string, error) {
	var s string
	if err := json.Unmarshal(r, &s); err != nil {
		return "", err
	}
	return s, nil
}
