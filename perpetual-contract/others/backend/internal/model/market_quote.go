package model

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MarketQuote 对应 public.market_quotes（Coinbase 指数价快照）。
type MarketQuote struct {
	Perp                  string    `gorm:"column:perp;primaryKey;size:42"`
	MarketName            string    `gorm:"column:market_name;not null"`
	ProductID             string    `gorm:"column:product_id;not null"`
	PriceUsd              string    `gorm:"column:price_usd;not null"`
	Open24h               string    `gorm:"column:open_24h;not null"`
	Volume24h             string    `gorm:"column:volume_24h;not null"`
	Low24h                string    `gorm:"column:low_24h;not null"`
	High24h               string    `gorm:"column:high_24h;not null"`
	PriceChange24h        string    `gorm:"column:price_change_24h;not null"`
	PriceChangePercent24h string    `gorm:"column:price_change_percent_24h;not null"`
	Source                string    `gorm:"column:source;not null"`
	UpdatedAt             time.Time `gorm:"column:updated_at;not null"`
}

func (MarketQuote) TableName() string { return "market_quotes" }

type MarketQuoteModel interface {
	Upsert(ctx context.Context, q *MarketQuote) error
	FindByPerp(ctx context.Context, perp string) (*MarketQuote, error)
}

type gormMarketQuoteModel struct {
	db *gorm.DB
}

func NewGormMarketQuoteModel(db *gorm.DB) MarketQuoteModel {
	return &gormMarketQuoteModel{db: db}
}

func (m *gormMarketQuoteModel) Upsert(ctx context.Context, q *MarketQuote) error {
	if q == nil {
		return nil
	}
	q.Perp = strings.ToLower(strings.TrimSpace(q.Perp))
	if q.UpdatedAt.IsZero() {
		q.UpdatedAt = time.Now().UTC()
	}
	if q.Source == "" {
		q.Source = "coinbase"
	}
	return m.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "perp"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"market_name", "product_id", "price_usd",
			"open_24h", "volume_24h", "low_24h", "high_24h",
			"price_change_24h", "price_change_percent_24h",
			"source", "updated_at",
		}),
	}).Create(q).Error
}

func (m *gormMarketQuoteModel) FindByPerp(ctx context.Context, perp string) (*MarketQuote, error) {
	perp = strings.ToLower(strings.TrimSpace(perp))
	var row MarketQuote
	err := m.db.WithContext(ctx).Where("perp = ?", perp).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}
