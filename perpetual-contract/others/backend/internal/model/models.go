package model

import (
	"context"
	"database/sql"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// ==================== Position 仓位模型 ====================

type Position struct {
	Id         int64     `db:"id"`
	Trader     string    `db:"trader"`
	Perp       string    `db:"perp"`
	Paper      string    `db:"paper"`
	Credit     string    `db:"credit"`
	UpdateTime time.Time `db:"update_time"`
}

type PositionModel interface {
	FindByTrader(ctx context.Context, trader string) ([]*Position, error)
	Upsert(ctx context.Context, position *Position) error
}

type defaultPositionModel struct {
	conn sqlx.SqlConn
}

func NewPositionModel(conn sqlx.SqlConn) PositionModel {
	return &defaultPositionModel{conn: conn}
}

func (m *defaultPositionModel) FindByTrader(ctx context.Context, trader string) ([]*Position, error) {
	var positions []*Position
	query := `SELECT * FROM positions WHERE trader = ? AND paper != '0'`
	err := m.conn.QueryRowsCtx(ctx, &positions, query, trader)
	return positions, err
}

func (m *defaultPositionModel) Upsert(ctx context.Context, position *Position) error {
	query := `INSERT INTO positions (trader, perp, paper, credit, update_time) 
		VALUES (?, ?, ?, ?, ?) 
		ON DUPLICATE KEY UPDATE paper = VALUES(paper), credit = VALUES(credit), update_time = VALUES(update_time)`
	_, err := m.conn.ExecCtx(ctx, query, position.Trader, position.Perp, position.Paper, position.Credit, time.Now())
	return err
}

// ==================== Deposit 存款记录模型 ====================

type Deposit struct {
	Id              int64     `db:"id"`
	TxHash          string    `db:"tx_hash"`
	LogIndex        int64     `db:"log_index"` // Transfer log index，与 tx_hash 联合唯一
	Trader          string    `db:"trader"`
	PrimaryAmount   string    `db:"primary_amount"`
	SecondaryAmount string    `db:"secondary_amount"`
	BlockNumber     int64     `db:"block_number"`
	CreateTime      time.Time `db:"create_time"`
}

type DepositModel interface {
	Insert(ctx context.Context, deposit *Deposit) (sql.Result, error)
	FindByTrader(ctx context.Context, trader string, page, pageSize int) ([]*Deposit, int64, error)
	FindByTxHashAndLogIndex(ctx context.Context, txHash string, logIndex uint) (*Deposit, error)
	MaxBlockNumber(ctx context.Context) (uint64, error)
}

type defaultDepositModel struct {
	conn sqlx.SqlConn
}

func NewDepositModel(conn sqlx.SqlConn) DepositModel {
	return &defaultDepositModel{conn: conn}
}

func (m *defaultDepositModel) Insert(ctx context.Context, deposit *Deposit) (sql.Result, error) {
	query := `INSERT INTO deposits (tx_hash, log_index, trader, primary_amount, secondary_amount, block_number, create_time) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	return m.conn.ExecCtx(ctx, query, deposit.TxHash, deposit.LogIndex, deposit.Trader, deposit.PrimaryAmount, deposit.SecondaryAmount, deposit.BlockNumber, deposit.CreateTime)
}

func (m *defaultDepositModel) FindByTrader(ctx context.Context, trader string, page, pageSize int) ([]*Deposit, int64, error) {
	var deposits []*Deposit
	var total int64

	countQuery := "SELECT COUNT(*) FROM deposits WHERE trader = ?"
	m.conn.QueryRowCtx(ctx, &total, countQuery, trader)

	offset := (page - 1) * pageSize
	query := `SELECT * FROM deposits WHERE trader = ? ORDER BY create_time DESC LIMIT ? OFFSET ?`
	err := m.conn.QueryRowsCtx(ctx, &deposits, query, trader, pageSize, offset)
	return deposits, total, err
}

func (m *defaultDepositModel) FindByTxHashAndLogIndex(ctx context.Context, txHash string, logIndex uint) (*Deposit, error) {
	var d Deposit
	err := m.conn.QueryRowCtx(ctx, &d, `SELECT * FROM deposits WHERE tx_hash = ? AND log_index = ? LIMIT 1`, txHash, int64(logIndex))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (m *defaultDepositModel) MaxBlockNumber(ctx context.Context) (uint64, error) {
	var maxBlock sql.NullInt64
	err := m.conn.QueryRowCtx(ctx, &maxBlock, `SELECT MAX(block_number) FROM deposits`)
	if err != nil || !maxBlock.Valid {
		return 0, err
	}
	return uint64(maxBlock.Int64), nil
}

// ==================== Withdraw 取款记录模型 ====================

type Withdraw struct {
	Id              int64     `db:"id"`
	TxHash          string    `db:"tx_hash"`
	Trader          string    `db:"trader"`
	PrimaryAmount   string    `db:"primary_amount"`
	SecondaryAmount string    `db:"secondary_amount"`
	Status          int       `db:"status"`
	BlockNumber     int64     `db:"block_number"`
	CreateTime      time.Time `db:"create_time"`
}

type WithdrawModel interface {
	Insert(ctx context.Context, withdraw *Withdraw) (sql.Result, error)
	FindByTrader(ctx context.Context, trader string, page, pageSize int) ([]*Withdraw, int64, error)
}

type defaultWithdrawModel struct {
	conn sqlx.SqlConn
}

func NewWithdrawModel(conn sqlx.SqlConn) WithdrawModel {
	return &defaultWithdrawModel{conn: conn}
}

func (m *defaultWithdrawModel) Insert(ctx context.Context, withdraw *Withdraw) (sql.Result, error) {
	query := `INSERT INTO withdraws (tx_hash, trader, primary_amount, secondary_amount, status, block_number, create_time) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	return m.conn.ExecCtx(ctx, query, withdraw.TxHash, withdraw.Trader, withdraw.PrimaryAmount, withdraw.SecondaryAmount, withdraw.Status, withdraw.BlockNumber, withdraw.CreateTime)
}

func (m *defaultWithdrawModel) FindByTrader(ctx context.Context, trader string, page, pageSize int) ([]*Withdraw, int64, error) {
	var withdraws []*Withdraw
	var total int64

	countQuery := "SELECT COUNT(*) FROM withdraws WHERE trader = ?"
	m.conn.QueryRowCtx(ctx, &total, countQuery, trader)

	offset := (page - 1) * pageSize
	query := `SELECT * FROM withdraws WHERE trader = ? ORDER BY create_time DESC LIMIT ? OFFSET ?`
	err := m.conn.QueryRowsCtx(ctx, &withdraws, query, trader, pageSize, offset)
	return withdraws, total, err
}

// ==================== FundingRate 资金费率模型 ====================

type FundingRate struct {
	Id         int64     `db:"id"`
	Perp       string    `db:"perp"`
	Rate       string    `db:"rate"`
	MarkPrice  string    `db:"mark_price"`
	IndexPrice string    `db:"index_price"`
	SettleTime time.Time `db:"settle_time"`
}

type FundingRateModel interface {
	Insert(ctx context.Context, rate *FundingRate) (sql.Result, error)
	FindByPerp(ctx context.Context, perp string, page, pageSize int) ([]*FundingRate, int64, error)
	GetLatest(ctx context.Context, perp string) (*FundingRate, error)
}

// ==================== Match Engine Event 撮合事件日志 ====================

const (
	EngineEventOrderAccepted = "order_accepted"
	EngineEventOrderCanceled = "order_canceled"
	EngineEventMatchEnqueued = "match_enqueued"
	EngineEventMatchSettled  = "match_settled"
	EngineEventMatchRollback = "match_rollback"
)

type EngineEvent struct {
	Seq        int64     `db:"id"`
	EventType  string    `db:"event_type"`
	OrderId    string    `db:"order_id"`
	MatchId    string    `db:"match_id"`
	Perp       string    `db:"perp"`
	Payload    string    `db:"payload"`
	NodeId     string    `db:"node_id"`
	CreateTime time.Time `db:"create_time"`
}

type EngineEventModel interface {
	Append(ctx context.Context, event *EngineEvent) (int64, error)
	LatestSeq(ctx context.Context) (int64, error)
	ListAfter(ctx context.Context, seq int64, limit int) ([]*EngineEvent, error)
	FindUnresolvedMatches(ctx context.Context, limit int) ([]*EngineEvent, error)
}

type defaultFundingRateModel struct {
	conn sqlx.SqlConn
}

func NewFundingRateModel(conn sqlx.SqlConn) FundingRateModel {
	return &defaultFundingRateModel{conn: conn}
}

func (m *defaultFundingRateModel) Insert(ctx context.Context, rate *FundingRate) (sql.Result, error) {
	query := `INSERT INTO funding_rates (perp, rate, mark_price, index_price, settle_time) VALUES (?, ?, ?, ?, ?)`
	return m.conn.ExecCtx(ctx, query, rate.Perp, rate.Rate, rate.MarkPrice, rate.IndexPrice, rate.SettleTime)
}

func (m *defaultFundingRateModel) FindByPerp(ctx context.Context, perp string, page, pageSize int) ([]*FundingRate, int64, error) {
	var rates []*FundingRate
	var total int64

	countQuery := "SELECT COUNT(*) FROM funding_rates WHERE perp = ?"
	m.conn.QueryRowCtx(ctx, &total, countQuery, perp)

	offset := (page - 1) * pageSize
	query := `SELECT * FROM funding_rates WHERE perp = ? ORDER BY settle_time DESC LIMIT ? OFFSET ?`
	err := m.conn.QueryRowsCtx(ctx, &rates, query, perp, pageSize, offset)
	return rates, total, err
}

func (m *defaultFundingRateModel) GetLatest(ctx context.Context, perp string) (*FundingRate, error) {
	var rate FundingRate
	query := `SELECT * FROM funding_rates WHERE perp = ? ORDER BY settle_time DESC LIMIT 1`
	err := m.conn.QueryRowCtx(ctx, &rate, query, perp)
	return &rate, err
}

// ==================== Liquidation 清算记录模型 ====================

type Liquidation struct {
	Id               int64     `db:"id"`
	TxHash           string    `db:"tx_hash"`
	Perp             string    `db:"perp"`
	Liquidator       string    `db:"liquidator"`
	LiquidatedTrader string    `db:"liquidated_trader"`
	PaperChange      string    `db:"paper_change"`
	CreditChange     string    `db:"credit_change"`
	InsuranceFee     string    `db:"insurance_fee"`
	BlockNumber      int64     `db:"block_number"`
	CreateTime       time.Time `db:"create_time"`
}

type LiquidationModel interface {
	Insert(ctx context.Context, liq *Liquidation) (sql.Result, error)
	Find(ctx context.Context, trader, perp string, page, pageSize int) ([]*Liquidation, int64, error)
}

type defaultLiquidationModel struct {
	conn sqlx.SqlConn
}

func NewLiquidationModel(conn sqlx.SqlConn) LiquidationModel {
	return &defaultLiquidationModel{conn: conn}
}

func (m *defaultLiquidationModel) Insert(ctx context.Context, liq *Liquidation) (sql.Result, error) {
	query := `INSERT INTO liquidations (tx_hash, perp, liquidator, liquidated_trader, paper_change, credit_change, insurance_fee, block_number, create_time) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	return m.conn.ExecCtx(ctx, query, liq.TxHash, liq.Perp, liq.Liquidator, liq.LiquidatedTrader, liq.PaperChange, liq.CreditChange, liq.InsuranceFee, liq.BlockNumber, liq.CreateTime)
}

func (m *defaultLiquidationModel) Find(ctx context.Context, trader, perp string, page, pageSize int) ([]*Liquidation, int64, error) {
	var liquidations []*Liquidation
	var total int64

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	if trader != "" {
		whereClause += " AND (liquidator = ? OR liquidated_trader = ?)"
		args = append(args, trader, trader)
	}
	if perp != "" {
		whereClause += " AND perp = ?"
		args = append(args, perp)
	}

	countQuery := "SELECT COUNT(*) FROM liquidations " + whereClause
	m.conn.QueryRowCtx(ctx, &total, countQuery, args...)

	offset := (page - 1) * pageSize
	query := "SELECT * FROM liquidations " + whereClause + " ORDER BY create_time DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, offset)
	err := m.conn.QueryRowsCtx(ctx, &liquidations, query, args...)
	return liquidations, total, err
}

// ==================== Market 市场模型 ====================

type Market struct {
	Id                   int64     `db:"id"`
	Address              string    `db:"address"`
	Name                 string    `db:"name"`
	InitialMarginRatio   string    `db:"initial_margin_ratio"`
	LiquidationThreshold string    `db:"liquidation_threshold"`
	LiquidationPriceOff  string    `db:"liquidation_price_off"`
	InsuranceFeeRate     string    `db:"insurance_fee_rate"`
	IsRegistered         bool      `db:"is_registered"`
	CreateTime           time.Time `db:"create_time"`
}

type MarketModel interface {
	FindAll(ctx context.Context) ([]*Market, error)
	FindByAddress(ctx context.Context, address string) (*Market, error)
	Upsert(ctx context.Context, market *Market) error
}

type defaultMarketModel struct {
	conn sqlx.SqlConn
}

func NewMarketModel(conn sqlx.SqlConn) MarketModel {
	return &defaultMarketModel{conn: conn}
}

func (m *defaultMarketModel) FindAll(ctx context.Context) ([]*Market, error) {
	var markets []*Market
	query := `SELECT * FROM markets WHERE is_registered = true`
	err := m.conn.QueryRowsCtx(ctx, &markets, query)
	return markets, err
}

func (m *defaultMarketModel) FindByAddress(ctx context.Context, address string) (*Market, error) {
	var market Market
	query := `SELECT * FROM markets WHERE address = ? LIMIT 1`
	err := m.conn.QueryRowCtx(ctx, &market, query, address)
	return &market, err
}

func (m *defaultMarketModel) Upsert(ctx context.Context, market *Market) error {
	query := `INSERT INTO markets (address, name, initial_margin_ratio, liquidation_threshold, liquidation_price_off, insurance_fee_rate, is_registered, create_time) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?) 
		ON DUPLICATE KEY UPDATE name = VALUES(name), initial_margin_ratio = VALUES(initial_margin_ratio), 
		liquidation_threshold = VALUES(liquidation_threshold), liquidation_price_off = VALUES(liquidation_price_off), 
		insurance_fee_rate = VALUES(insurance_fee_rate), is_registered = VALUES(is_registered)`
	_, err := m.conn.ExecCtx(ctx, query, market.Address, market.Name, market.InitialMarginRatio, market.LiquidationThreshold, market.LiquidationPriceOff, market.InsuranceFeeRate, market.IsRegistered, time.Now())
	return err
}

// ==================== Kline K线模型 ====================

type Kline struct {
	Id       int64     `db:"id"`
	Perp     string    `db:"perp"`
	Interval string    `db:"interval_type"` // 1m, 5m, 15m, 1h, 4h, 1d
	OpenTime time.Time `db:"open_time"`
	Open     string    `db:"open_price"`
	High     string    `db:"high_price"`
	Low      string    `db:"low_price"`
	Close    string    `db:"close_price"`
	Volume   string    `db:"volume"`
}

type KlineModel interface {
	Upsert(ctx context.Context, kline *Kline) error
	Find(ctx context.Context, perp, interval string, startTime, endTime time.Time, limit int) ([]*Kline, error)
}

type defaultKlineModel struct {
	conn sqlx.SqlConn
}

func NewKlineModel(conn sqlx.SqlConn) KlineModel {
	return &defaultKlineModel{conn: conn}
}

func (m *defaultKlineModel) Upsert(ctx context.Context, kline *Kline) error {
	query := `INSERT INTO klines (perp, interval_type, open_time, open_price, high_price, low_price, close_price, volume) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?) 
		ON DUPLICATE KEY UPDATE high_price = GREATEST(high_price, VALUES(high_price)), 
		low_price = LEAST(low_price, VALUES(low_price)), close_price = VALUES(close_price), 
		volume = volume + VALUES(volume)`
	_, err := m.conn.ExecCtx(ctx, query, kline.Perp, kline.Interval, kline.OpenTime, kline.Open, kline.High, kline.Low, kline.Close, kline.Volume)
	return err
}

func (m *defaultKlineModel) Find(ctx context.Context, perp, interval string, startTime, endTime time.Time, limit int) ([]*Kline, error) {
	var klines []*Kline
	query := `SELECT * FROM klines WHERE perp = ? AND interval_type = ? AND open_time >= ? AND open_time <= ? ORDER BY open_time ASC LIMIT ?`
	err := m.conn.QueryRowsCtx(ctx, &klines, query, perp, interval, startTime, endTime, limit)
	return klines, err
}
