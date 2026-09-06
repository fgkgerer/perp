package model

import (
	"context"
	"math"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SpotKline 现货指数 K 线（按 perp + 周期 + 开盘时间唯一）。
type SpotKline struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement"`
	Perp         string    `gorm:"column:perp;not null;size:42"`
	IntervalType string    `gorm:"column:interval_type;not null;size:8"`
	OpenTime     time.Time `gorm:"column:open_time;not null"`
	OpenPrice    string    `gorm:"column:open_price;not null"`
	HighPrice    string    `gorm:"column:high_price;not null"`
	LowPrice     string    `gorm:"column:low_price;not null"`
	ClosePrice   string    `gorm:"column:close_price;not null"`
	Volume       string    `gorm:"column:volume;not null;default:0"`
	UpdatedAt    time.Time `gorm:"column:updated_at;not null"`
}

func (SpotKline) TableName() string { return "spot_klines" }

type SpotKlineModel interface {
	UpsertBar(ctx context.Context, bar *SpotKline) error
	FindBars(ctx context.Context, perp, interval string, start, end time.Time, limit int) ([]SpotKline, error)
	CountBars(ctx context.Context, perp, interval string, start, end time.Time) (int64, error)
}

type gormSpotKlineModel struct {
	db *gorm.DB
}

func NewGormSpotKlineModel(db *gorm.DB) SpotKlineModel {
	return &gormSpotKlineModel{db: db}
}

func parsePriceFloat(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) || f <= 0 {
		return 0, false
	}
	return f, true
}

func formatPriceFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func (m *gormSpotKlineModel) UpsertBar(ctx context.Context, bar *SpotKline) error {
	if bar == nil {
		return nil
	}
	bar.Perp = strings.ToLower(strings.TrimSpace(bar.Perp))
	bar.IntervalType = strings.TrimSpace(bar.IntervalType)
	if bar.Perp == "" || bar.IntervalType == "" || bar.OpenTime.IsZero() {
		return nil
	}
	if bar.Volume == "" {
		bar.Volume = "0"
	}
	now := time.Now().UTC()
	if bar.UpdatedAt.IsZero() {
		bar.UpdatedAt = now
	}

	var existing SpotKline
	err := m.db.WithContext(ctx).
		Where("perp = ? AND interval_type = ? AND open_time = ?", bar.Perp, bar.IntervalType, bar.OpenTime.UTC()).
		First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return m.db.WithContext(ctx).Create(bar).Error
	}
	if err != nil {
		return err
	}

	o, okO := parsePriceFloat(bar.OpenPrice)
	h, okH := parsePriceFloat(bar.HighPrice)
	l, okL := parsePriceFloat(bar.LowPrice)
	c, okC := parsePriceFloat(bar.ClosePrice)
	if !okO || !okH || !okL || !okC {
		return nil
	}
	eo, _ := parsePriceFloat(existing.OpenPrice)
	eh, _ := parsePriceFloat(existing.HighPrice)
	el, _ := parsePriceFloat(existing.LowPrice)
	if eo <= 0 {
		eo = o
	}
	eh = math.Max(eh, h)
	el = math.Min(el, l)
	if el <= 0 || el > eh {
		el = l
	}

	merged := SpotKline{
		ID:           existing.ID,
		Perp:         existing.Perp,
		IntervalType: existing.IntervalType,
		OpenTime:     existing.OpenTime,
		OpenPrice:    formatPriceFloat(eo),
		HighPrice:    formatPriceFloat(eh),
		LowPrice:     formatPriceFloat(el),
		ClosePrice:   formatPriceFloat(c),
		Volume:       existing.Volume,
		UpdatedAt:    now,
	}
	return m.db.WithContext(ctx).Save(&merged).Error
}

func (m *gormSpotKlineModel) FindBars(
	ctx context.Context,
	perp, interval string,
	start, end time.Time,
	limit int,
) ([]SpotKline, error) {
	perp = strings.ToLower(strings.TrimSpace(perp))
	interval = strings.TrimSpace(interval)
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	var rows []SpotKline
	err := m.db.WithContext(ctx).
		Where("perp = ? AND interval_type = ? AND open_time >= ? AND open_time <= ?",
			perp, interval, start.UTC(), end.UTC()).
		Order("open_time ASC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

func (m *gormSpotKlineModel) CountBars(
	ctx context.Context,
	perp, interval string,
	start, end time.Time,
) (int64, error) {
	perp = strings.ToLower(strings.TrimSpace(perp))
	var n int64
	err := m.db.WithContext(ctx).Model(&SpotKline{}).
		Where("perp = ? AND interval_type = ? AND open_time >= ? AND open_time <= ?",
			perp, interval, start.UTC(), end.UTC()).
		Count(&n).Error
	return n, err
}

// BulkUpsertBars 批量写入历史 K 线（Binance 回填）；冲突时更新高低收。
func BulkUpsertBars(ctx context.Context, db *gorm.DB, bars []SpotKline) error {
	if db == nil || len(bars) == 0 {
		return nil
	}
	now := time.Now().UTC()
	for i := range bars {
		bars[i].Perp = strings.ToLower(strings.TrimSpace(bars[i].Perp))
		if bars[i].Volume == "" {
			bars[i].Volume = "0"
		}
		if bars[i].UpdatedAt.IsZero() {
			bars[i].UpdatedAt = now
		}
	}
	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "perp"},
			{Name: "interval_type"},
			{Name: "open_time"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"high_price", "low_price", "close_price", "volume", "updated_at",
		}),
	}).CreateInBatches(bars, 200).Error
}
