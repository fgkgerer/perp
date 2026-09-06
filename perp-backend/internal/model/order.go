package model

import (
	"context"
	"database/sql"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// 订单状态
const (
	OrderStatusPending     = 0 // 待处理
	OrderStatusPartialFill = 1 // 部分成交
	OrderStatusFilled      = 2 // 完全成交
	OrderStatusCancelled   = 3 // 已取消
)

// Order 订单数据模型
type Order struct {
	Id           int64     `db:"id"`
	OrderId      string    `db:"order_id"`      // 订单ID (hash)
	ChainId      int64     `db:"chain_id"`      // EVM chain id（Supabase 旧表必填）
	Perp         string    `db:"perp"`          // 永续合约地址
	Signer       string    `db:"signer"`        // 签名者地址
	PaperAmount  string    `db:"paper_amount"`  // 仓位数量
	CreditAmount string    `db:"credit_amount"` // 资金数量
	MakerFeeRate string    `db:"maker_fee_rate"`
	TakerFeeRate string    `db:"taker_fee_rate"`
	Expiration   int64     `db:"expiration"`
	Nonce        int64     `db:"nonce"`
	Signature    string    `db:"signature"`
	Status       int       `db:"status"`
	FilledAmount string    `db:"filled_amount"` // 已成交数量
	CreateTime   time.Time `db:"create_time"`
	UpdateTime   time.Time `db:"update_time"`
}

// OrderModel 订单模型接口
type OrderModel interface {
	Insert(ctx context.Context, order *Order) (sql.Result, error)
	FindOne(ctx context.Context, orderId string) (*Order, error)
	FindByTrader(ctx context.Context, signer, perp string, status int, page, pageSize int) ([]*Order, int64, error)
	Update(ctx context.Context, order *Order) error
	UpdateStatus(ctx context.Context, orderId string, status int, filledAmount string) error
	FindPendingOrders(ctx context.Context, perp string, limit int) ([]*Order, error)
}

// defaultOrderModel 默认订单模型实现
type defaultOrderModel struct {
	conn sqlx.SqlConn
}

// NewOrderModel 创建订单模型
func NewOrderModel(conn sqlx.SqlConn) OrderModel {
	return &defaultOrderModel{conn: conn}
}

// Insert 插入订单
func (m *defaultOrderModel) Insert(ctx context.Context, order *Order) (sql.Result, error) {
	query := `INSERT INTO orders (order_id, perp, signer, paper_amount, credit_amount, 
		maker_fee_rate, taker_fee_rate, expiration, nonce, signature, status, filled_amount, 
		create_time, update_time) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	return m.conn.ExecCtx(ctx, query,
		order.OrderId, order.Perp, order.Signer, order.PaperAmount, order.CreditAmount,
		order.MakerFeeRate, order.TakerFeeRate, order.Expiration, order.Nonce,
		order.Signature, order.Status, order.FilledAmount,
		order.CreateTime, order.UpdateTime)
}

// FindOne 根据订单ID查询
func (m *defaultOrderModel) FindOne(ctx context.Context, orderId string) (*Order, error) {
	var order Order
	query := `SELECT * FROM orders WHERE order_id = ? LIMIT 1`
	err := m.conn.QueryRowCtx(ctx, &order, query, orderId)
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// FindByTrader 查询交易者订单列表
func (m *defaultOrderModel) FindByTrader(ctx context.Context, signer, perp string, status int, page, pageSize int) ([]*Order, int64, error) {
	var orders []*Order
	var total int64

	// 构建查询条件
	whereClause := "WHERE signer = ?"
	args := []interface{}{signer}

	if perp != "" {
		whereClause += " AND perp = ?"
		args = append(args, perp)
	}
	if status >= 0 {
		whereClause += " AND status = ?"
		args = append(args, status)
	}

	// 查询总数
	countQuery := "SELECT COUNT(*) FROM orders " + whereClause
	err := m.conn.QueryRowCtx(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// 查询列表
	offset := (page - 1) * pageSize
	listQuery := "SELECT * FROM orders " + whereClause + " ORDER BY create_time DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, offset)

	err = m.conn.QueryRowsCtx(ctx, &orders, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// Update 更新订单
func (m *defaultOrderModel) Update(ctx context.Context, order *Order) error {
	query := `UPDATE orders SET status = ?, filled_amount = ?, update_time = ? WHERE order_id = ?`
	_, err := m.conn.ExecCtx(ctx, query, order.Status, order.FilledAmount, time.Now(), order.OrderId)
	return err
}

// UpdateStatus 更新订单状态
func (m *defaultOrderModel) UpdateStatus(ctx context.Context, orderId string, status int, filledAmount string) error {
	query := `UPDATE orders SET status = ?, filled_amount = ?, update_time = ? WHERE order_id = ?`
	_, err := m.conn.ExecCtx(ctx, query, status, filledAmount, time.Now(), orderId)
	return err
}

// FindPendingOrders 查询待处理订单
func (m *defaultOrderModel) FindPendingOrders(ctx context.Context, perp string, limit int) ([]*Order, error) {
	var orders []*Order
	query := `SELECT * FROM orders WHERE perp = ? AND status IN (?, ?) AND expiration > ? 
		ORDER BY create_time ASC LIMIT ?`
	err := m.conn.QueryRowsCtx(ctx, &orders, query, perp, OrderStatusPending, OrderStatusPartialFill, time.Now().Unix(), limit)
	if err != nil {
		return nil, err
	}
	return orders, nil
}
