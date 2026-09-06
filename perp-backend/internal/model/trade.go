package model

import (
	"context"
	"database/sql"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// Trade 成交记录数据模型
type Trade struct {
	Id           int64     `db:"id"`
	TradeId      string    `db:"trade_id"`
	Perp         string    `db:"perp"`
	TakerOrderId string    `db:"taker_order_id"`
	MakerOrderId string    `db:"maker_order_id"`
	Taker        string    `db:"taker"`
	Maker        string    `db:"maker"`
	PaperAmount  string    `db:"paper_amount"`
	Price        string    `db:"price"`
	TakerFee     string    `db:"taker_fee"`
	MakerFee     string    `db:"maker_fee"`
	TxHash       string    `db:"tx_hash"`
	BlockNumber  int64     `db:"block_number"`
	CreateTime   time.Time `db:"create_time"`
}

// TradeModel 成交记录模型接口
type TradeModel interface {
	Insert(ctx context.Context, trade *Trade) (sql.Result, error)
	FindByTrader(ctx context.Context, trader, perp string, page, pageSize int) ([]*Trade, int64, error)
	FindRecent(ctx context.Context, perp string, limit int) ([]*Trade, error)
	GetVolume24h(ctx context.Context, perp string) (string, error)
}

// defaultTradeModel 默认成交记录模型实现
type defaultTradeModel struct {
	conn sqlx.SqlConn
}

// NewTradeModel 创建成交记录模型
func NewTradeModel(conn sqlx.SqlConn) TradeModel {
	return &defaultTradeModel{conn: conn}
}

// Insert 插入成交记录
func (m *defaultTradeModel) Insert(ctx context.Context, trade *Trade) (sql.Result, error) {
	query := `INSERT INTO trades (trade_id, perp, taker_order_id, maker_order_id, taker, maker, 
		paper_amount, price, taker_fee, maker_fee, tx_hash, block_number, create_time) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	return m.conn.ExecCtx(ctx, query,
		trade.TradeId, trade.Perp, trade.TakerOrderId, trade.MakerOrderId,
		trade.Taker, trade.Maker, trade.PaperAmount, trade.Price,
		trade.TakerFee, trade.MakerFee, trade.TxHash, trade.BlockNumber, trade.CreateTime)
}

// FindByTrader 查询交易者成交记录
func (m *defaultTradeModel) FindByTrader(ctx context.Context, trader, perp string, page, pageSize int) ([]*Trade, int64, error) {
	var trades []*Trade
	var total int64

	// 构建查询条件
	whereClause := "WHERE (taker = ? OR maker = ?)"
	args := []interface{}{trader, trader}

	if perp != "" {
		whereClause += " AND perp = ?"
		args = append(args, perp)
	}

	// 查询总数
	countQuery := "SELECT COUNT(*) FROM trades " + whereClause
	err := m.conn.QueryRowCtx(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// 查询列表
	offset := (page - 1) * pageSize
	listQuery := "SELECT * FROM trades " + whereClause + " ORDER BY create_time DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, offset)

	err = m.conn.QueryRowsCtx(ctx, &trades, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return trades, total, nil
}

// FindRecent 查询最近成交
func (m *defaultTradeModel) FindRecent(ctx context.Context, perp string, limit int) ([]*Trade, error) {
	var trades []*Trade
	query := `SELECT * FROM trades WHERE perp = ? ORDER BY create_time DESC LIMIT ?`
	err := m.conn.QueryRowsCtx(ctx, &trades, query, perp, limit)
	if err != nil {
		return nil, err
	}
	return trades, nil
}

// GetVolume24h 获取24小时成交量
func (m *defaultTradeModel) GetVolume24h(ctx context.Context, perp string) (string, error) {
	var volume string
	yesterday := time.Now().Add(-24 * time.Hour)
	query := `SELECT COALESCE(SUM(ABS(paper_amount)), '0') FROM trades WHERE perp = ? AND create_time >= ?`
	err := m.conn.QueryRowCtx(ctx, &volume, query, perp, yesterday)
	if err != nil {
		return "0", err
	}
	return volume, nil
}
