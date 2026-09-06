package model

import (
	"context"
	"database/sql"
	"errors"
	"math/big"
	"strings"
	"time"

	"gorm.io/gorm"
)

// depositRow / ledgerBalanceRow 与 public.deposits、public.ledger_balances 对齐。
type depositRow struct {
	ID              int64     `gorm:"column:id;primaryKey;autoIncrement"`
	TxHash          string    `gorm:"column:tx_hash;size:66;not null"`
	LogIndex        int64     `gorm:"column:log_index;not null"`
	Trader          string    `gorm:"column:trader;size:42;not null;index"`
	PrimaryAmount   string    `gorm:"column:primary_amount;not null"`
	SecondaryAmount string    `gorm:"column:secondary_amount;not null"`
	BlockNumber     int64     `gorm:"column:block_number;not null"`
	CreateTime      time.Time `gorm:"column:create_time;not null"`
}

func (depositRow) TableName() string { return "deposits" }

type ledgerBalanceRow struct {
	Trader         string    `gorm:"column:trader;primaryKey;size:42"`
	PrimaryBalance string    `gorm:"column:primary_balance;not null"`
	UpdatedAt      time.Time `gorm:"column:updated_at;not null"`
}

func (ledgerBalanceRow) TableName() string { return "ledger_balances" }

type gormDepositModel struct {
	db *gorm.DB
}

func NewGormDepositModel(db *gorm.DB) DepositModel {
	return &gormDepositModel{db: db}
}

func (m *gormDepositModel) Insert(ctx context.Context, deposit *Deposit) (sql.Result, error) {
	row := depositRow{
		TxHash:          deposit.TxHash,
		LogIndex:        deposit.LogIndex,
		Trader:          deposit.Trader,
		PrimaryAmount:   deposit.PrimaryAmount,
		SecondaryAmount: deposit.SecondaryAmount,
		BlockNumber:     deposit.BlockNumber,
		CreateTime:      deposit.CreateTime,
	}
	if row.CreateTime.IsZero() {
		row.CreateTime = time.Now().UTC()
	}
	err := m.db.WithContext(ctx).Create(&row).Error
	if err != nil {
		return nil, err
	}
	deposit.Id = row.ID
	return gormResult{rows: 1, lastID: row.ID}, nil
}

func (m *gormDepositModel) FindByTrader(ctx context.Context, trader string, page, pageSize int) ([]*Deposit, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	trader = strings.ToLower(strings.TrimSpace(trader))
	var total int64
	if err := m.db.WithContext(ctx).Model(&depositRow{}).Where("trader = ?", trader).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []depositRow
	offset := (page - 1) * pageSize
	err := m.db.WithContext(ctx).
		Where("trader = ?", trader).
		Order("create_time DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	out := make([]*Deposit, 0, len(rows))
	for i := range rows {
		out = append(out, depositRowToModel(&rows[i]))
	}
	return out, total, nil
}

func (m *gormDepositModel) MaxBlockNumber(ctx context.Context) (uint64, error) {
	var maxBlock *int64
	err := m.db.WithContext(ctx).Model(&depositRow{}).
		Select("MAX(block_number)").
		Scan(&maxBlock).Error
	if err != nil {
		return 0, err
	}
	if maxBlock == nil {
		return 0, nil
	}
	return uint64(*maxBlock), nil
}

func (m *gormDepositModel) FindByTxHashAndLogIndex(ctx context.Context, txHash string, logIndex uint) (*Deposit, error) {
	var row depositRow
	err := m.db.WithContext(ctx).
		Where("tx_hash = ? AND log_index = ?", txHash, int64(logIndex)).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return depositRowToModel(&row), nil
}

type gormLedgerBalanceModel struct {
	db *gorm.DB
}

func NewGormLedgerBalanceModel(db *gorm.DB) LedgerBalanceModel {
	return &gormLedgerBalanceModel{db: db}
}

func (m *gormLedgerBalanceModel) GetPrimary(ctx context.Context, traderLower string) (*big.Int, error) {
	key := normalizeTraderKey(traderLower)
	var row ledgerBalanceRow
	err := m.db.WithContext(ctx).Where("trader = ?", key).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return big.NewInt(0), nil
	}
	if err != nil {
		return nil, err
	}
	amt, ok := new(big.Int).SetString(row.PrimaryBalance, 10)
	if !ok {
		return big.NewInt(0), nil
	}
	return amt, nil
}

func (m *gormLedgerBalanceModel) AddPrimary(ctx context.Context, traderLower string, delta *big.Int) error {
	if delta == nil || delta.Sign() <= 0 {
		return nil
	}
	key := normalizeTraderKey(traderLower)
	cur, err := m.GetPrimary(ctx, key)
	if err != nil {
		return err
	}
	sum := new(big.Int).Add(cur, delta)
	now := time.Now().UTC()
	return m.db.WithContext(ctx).Exec(`
INSERT INTO ledger_balances (trader, primary_balance, updated_at)
VALUES (?, ?, ?)
ON CONFLICT (trader) DO UPDATE
SET primary_balance = EXCLUDED.primary_balance, updated_at = EXCLUDED.updated_at`,
		key, sum.String(), now,
	).Error
}

func depositRowToModel(r *depositRow) *Deposit {
	return &Deposit{
		Id:              r.ID,
		TxHash:          r.TxHash,
		LogIndex:        r.LogIndex,
		Trader:          r.Trader,
		PrimaryAmount:   r.PrimaryAmount,
		SecondaryAmount: r.SecondaryAmount,
		BlockNumber:     r.BlockNumber,
		CreateTime:      r.CreateTime,
	}
}

type gormResult struct {
	rows   int64
	lastID int64
}

func (r gormResult) LastInsertId() (int64, error) { return r.lastID, nil }
func (r gormResult) RowsAffected() (int64, error) { return r.rows, nil }

func isDuplicateKeyErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique constraint")
}

// IsDuplicateKeyErr Postgres 唯一约束冲突（扫链重复入账时忽略）。
func IsDuplicateKeyErr(err error) bool {
	return isDuplicateKeyErr(err)
}

// --- orders / trades (GORM) ---

type orderRow struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement"`
	OrderId      string    `gorm:"column:order_id;size:66;not null;uniqueIndex"`
	ChainId      int64     `gorm:"column:chain_id;not null"`
	Perp         string    `gorm:"column:perp;size:42;not null;index"`
	PerpAddress  string    `gorm:"column:perp_address;size:42;not null"` // Supabase 旧表列名
	Signer       string    `gorm:"column:signer;size:42;not null;index"`
	PaperAmount  string    `gorm:"column:paper_amount;not null"`
	CreditAmount string    `gorm:"column:credit_amount;not null"`
	MakerFeeRate string    `gorm:"column:maker_fee_rate;not null"`
	TakerFeeRate string    `gorm:"column:taker_fee_rate;not null"`
	Expiration   int64     `gorm:"column:expiration;not null"`
	Nonce        int64     `gorm:"column:nonce;not null"`
	Signature    string    `gorm:"column:signature;not null"`
	Status       int       `gorm:"column:status;not null;index"`
	FilledAmount string    `gorm:"column:filled_amount;not null"`
	CreateTime   time.Time `gorm:"column:create_time;not null"`
	UpdateTime   time.Time `gorm:"column:update_time;not null"`
}

func (orderRow) TableName() string { return "orders" }

type tradeRow struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement"`
	TradeId      string    `gorm:"column:trade_id;size:64;not null;uniqueIndex"`
	Perp         string    `gorm:"column:perp;size:42;not null;index"`
	TakerOrderId string    `gorm:"column:taker_order_id;size:66;not null"`
	MakerOrderId string    `gorm:"column:maker_order_id;size:66;not null"`
	Taker        string    `gorm:"column:taker;size:42;not null;index"`
	Maker        string    `gorm:"column:maker;size:42;not null;index"`
	PaperAmount  string    `gorm:"column:paper_amount;not null"`
	Price        string    `gorm:"column:price;not null"`
	TakerFee     string    `gorm:"column:taker_fee;not null"`
	MakerFee     string    `gorm:"column:maker_fee;not null"`
	TxHash       string    `gorm:"column:tx_hash;size:66;not null"`
	BlockNumber  int64     `gorm:"column:block_number;not null"`
	CreateTime   time.Time `gorm:"column:create_time;not null"`
}

func (tradeRow) TableName() string { return "trades" }

type engineEventRow struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement"`
	EventType  string    `gorm:"column:event_type;size:64;not null;index"`
	OrderId    string    `gorm:"column:order_id;size:66;not null;index"`
	MatchId    string    `gorm:"column:match_id;size:80;not null;index"`
	Perp       string    `gorm:"column:perp;size:42;not null;index"`
	Payload    string    `gorm:"column:payload;type:jsonb;not null"`
	NodeId     string    `gorm:"column:node_id;size:128;not null"`
	CreateTime time.Time `gorm:"column:create_time;not null"`
}

func (engineEventRow) TableName() string { return "engine_events" }

type gormOrderModel struct {
	db *gorm.DB
}

func NewGormOrderModel(db *gorm.DB) OrderModel {
	return &gormOrderModel{db: db}
}

func orderPerp(r *orderRow) string {
	if r.Perp != "" {
		return r.Perp
	}
	return r.PerpAddress
}

func orderToRow(o *Order) orderRow {
	perp := o.Perp
	row := orderRow{
		OrderId:      o.OrderId,
		ChainId:      o.ChainId,
		Perp:         perp,
		PerpAddress:  perp,
		Signer:       o.Signer,
		PaperAmount:  o.PaperAmount,
		CreditAmount: o.CreditAmount,
		MakerFeeRate: o.MakerFeeRate,
		TakerFeeRate: o.TakerFeeRate,
		Expiration:   o.Expiration,
		Nonce:        o.Nonce,
		Signature:    o.Signature,
		Status:       o.Status,
		FilledAmount: o.FilledAmount,
		CreateTime:   o.CreateTime,
		UpdateTime:   o.UpdateTime,
	}
	if row.FilledAmount == "" {
		row.FilledAmount = "0"
	}
	if row.CreateTime.IsZero() {
		row.CreateTime = time.Now().UTC()
	}
	if row.UpdateTime.IsZero() {
		row.UpdateTime = row.CreateTime
	}
	if row.ChainId == 0 {
		row.ChainId = 11155111
	}
	return row
}

func orderRowToModel(r *orderRow) *Order {
	return &Order{
		Id:           r.ID,
		OrderId:      r.OrderId,
		ChainId:      r.ChainId,
		Perp:         orderPerp(r),
		Signer:       r.Signer,
		PaperAmount:  r.PaperAmount,
		CreditAmount: r.CreditAmount,
		MakerFeeRate: r.MakerFeeRate,
		TakerFeeRate: r.TakerFeeRate,
		Expiration:   r.Expiration,
		Nonce:        r.Nonce,
		Signature:    r.Signature,
		Status:       r.Status,
		FilledAmount: r.FilledAmount,
		CreateTime:   r.CreateTime,
		UpdateTime:   r.UpdateTime,
	}
}

func (m *gormOrderModel) Insert(ctx context.Context, order *Order) (sql.Result, error) {
	row := orderToRow(order)
	err := m.db.WithContext(ctx).Create(&row).Error
	if err != nil {
		return nil, err
	}
	order.Id = row.ID
	return gormResult{rows: 1, lastID: row.ID}, nil
}

func (m *gormOrderModel) FindOne(ctx context.Context, orderId string) (*Order, error) {
	var row orderRow
	err := m.db.WithContext(ctx).Where("order_id = ?", orderId).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return orderRowToModel(&row), nil
}

func (m *gormOrderModel) FindByTrader(ctx context.Context, signer, perp string, status int, page, pageSize int) ([]*Order, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	q := m.db.WithContext(ctx).Model(&orderRow{}).Where("signer = ?", signer)
	if perp != "" {
		q = q.Where("(perp = ? OR perp_address = ?)", perp, perp)
	}
	if status >= 0 {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []orderRow
	offset := (page - 1) * pageSize
	err := q.Order("create_time DESC").Limit(pageSize).Offset(offset).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	out := make([]*Order, 0, len(rows))
	for i := range rows {
		out = append(out, orderRowToModel(&rows[i]))
	}
	return out, total, nil
}

func (m *gormOrderModel) Update(ctx context.Context, order *Order) error {
	return m.db.WithContext(ctx).Model(&orderRow{}).
		Where("order_id = ?", order.OrderId).
		Updates(map[string]interface{}{
			"status":        order.Status,
			"filled_amount": order.FilledAmount,
			"update_time":   time.Now().UTC(),
		}).Error
}

func (m *gormOrderModel) UpdateStatus(ctx context.Context, orderId string, status int, filledAmount string) error {
	return m.db.WithContext(ctx).Model(&orderRow{}).
		Where("order_id = ?", orderId).
		Updates(map[string]interface{}{
			"status":        status,
			"filled_amount": filledAmount,
			"update_time":   time.Now().UTC(),
		}).Error
}

func (m *gormOrderModel) FindPendingOrders(ctx context.Context, perp string, limit int) ([]*Order, error) {
	if limit <= 0 {
		limit = 100
	}
	var rows []orderRow
	err := m.db.WithContext(ctx).
		Where("(perp = ? OR perp_address = ?) AND status IN ? AND expiration > ?", perp, perp, []int{OrderStatusPending, OrderStatusPartialFill}, time.Now().Unix()).
		Order("create_time ASC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*Order, 0, len(rows))
	for i := range rows {
		out = append(out, orderRowToModel(&rows[i]))
	}
	return out, nil
}

type gormTradeModel struct {
	db *gorm.DB
}

func NewGormTradeModel(db *gorm.DB) TradeModel {
	return &gormTradeModel{db: db}
}

func (m *gormTradeModel) Insert(ctx context.Context, trade *Trade) (sql.Result, error) {
	row := tradeRow{
		TradeId:      trade.TradeId,
		Perp:         trade.Perp,
		TakerOrderId: trade.TakerOrderId,
		MakerOrderId: trade.MakerOrderId,
		Taker:        trade.Taker,
		Maker:        trade.Maker,
		PaperAmount:  trade.PaperAmount,
		Price:        trade.Price,
		TakerFee:     trade.TakerFee,
		MakerFee:     trade.MakerFee,
		TxHash:       trade.TxHash,
		BlockNumber:  trade.BlockNumber,
		CreateTime:   trade.CreateTime,
	}
	if row.CreateTime.IsZero() {
		row.CreateTime = time.Now().UTC()
	}
	err := m.db.WithContext(ctx).Create(&row).Error
	if err != nil {
		return nil, err
	}
	trade.Id = row.ID
	return gormResult{rows: 1, lastID: row.ID}, nil
}

func (m *gormTradeModel) FindByTrader(ctx context.Context, trader, perp string, page, pageSize int) ([]*Trade, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	q := m.db.WithContext(ctx).Model(&tradeRow{}).Where("taker = ? OR maker = ?", trader, trader)
	if perp != "" {
		q = q.Where("perp = ?", perp)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []tradeRow
	offset := (page - 1) * pageSize
	err := q.Order("create_time DESC").Limit(pageSize).Offset(offset).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	out := make([]*Trade, 0, len(rows))
	for i := range rows {
		out = append(out, tradeRowToModel(&rows[i]))
	}
	return out, total, nil
}

func tradeRowToModel(r *tradeRow) *Trade {
	return &Trade{
		Id:           r.ID,
		TradeId:      r.TradeId,
		Perp:         r.Perp,
		TakerOrderId: r.TakerOrderId,
		MakerOrderId: r.MakerOrderId,
		Taker:        r.Taker,
		Maker:        r.Maker,
		PaperAmount:  r.PaperAmount,
		Price:        r.Price,
		TakerFee:     r.TakerFee,
		MakerFee:     r.MakerFee,
		TxHash:       r.TxHash,
		BlockNumber:  r.BlockNumber,
		CreateTime:   r.CreateTime,
	}
}

func (m *gormTradeModel) FindRecent(ctx context.Context, perp string, limit int) ([]*Trade, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []tradeRow
	err := m.db.WithContext(ctx).
		Where("perp = ?", perp).
		Order("create_time DESC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*Trade, 0, len(rows))
	for i := range rows {
		out = append(out, tradeRowToModel(&rows[i]))
	}
	return out, nil
}

func (m *gormTradeModel) GetVolume24h(ctx context.Context, perp string) (string, error) {
	yesterday := time.Now().Add(-24 * time.Hour).UTC()
	var volume string
	err := m.db.WithContext(ctx).Model(&tradeRow{}).
		Select("COALESCE(SUM(ABS(CAST(paper_amount AS NUMERIC))), 0)::TEXT").
		Where("perp = ? AND create_time >= ?", perp, yesterday).
		Scan(&volume).Error
	if err != nil {
		return "0", err
	}
	if volume == "" {
		return "0", nil
	}
	return volume, nil
}

type gormEngineEventModel struct {
	db *gorm.DB
}

func NewGormEngineEventModel(db *gorm.DB) EngineEventModel {
	return &gormEngineEventModel{db: db}
}

func engineEventToRow(e *EngineEvent) engineEventRow {
	row := engineEventRow{
		EventType:  e.EventType,
		OrderId:    e.OrderId,
		MatchId:    e.MatchId,
		Perp:       e.Perp,
		Payload:    e.Payload,
		NodeId:     e.NodeId,
		CreateTime: e.CreateTime,
	}
	if row.CreateTime.IsZero() {
		row.CreateTime = time.Now().UTC()
	}
	return row
}

func engineEventRowToModel(r *engineEventRow) *EngineEvent {
	return &EngineEvent{
		Seq:        r.ID,
		EventType:  r.EventType,
		OrderId:    r.OrderId,
		MatchId:    r.MatchId,
		Perp:       r.Perp,
		Payload:    r.Payload,
		NodeId:     r.NodeId,
		CreateTime: r.CreateTime,
	}
}

func (m *gormEngineEventModel) Append(ctx context.Context, event *EngineEvent) (int64, error) {
	row := engineEventToRow(event)
	var id int64
	err := m.db.WithContext(ctx).Raw(`
INSERT INTO engine_events (event_type, order_id, match_id, perp, payload, node_id, create_time)
VALUES (?, ?, ?, ?, ?::jsonb, ?, ?)
RETURNING id`,
		row.EventType, row.OrderId, row.MatchId, row.Perp, row.Payload, row.NodeId, row.CreateTime,
	).Scan(&id).Error
	if err != nil {
		return 0, err
	}
	event.Seq = id
	return id, nil
}

func (m *gormEngineEventModel) LatestSeq(ctx context.Context) (int64, error) {
	var seq *int64
	err := m.db.WithContext(ctx).Model(&engineEventRow{}).Select("MAX(id)").Scan(&seq).Error
	if err != nil || seq == nil {
		return 0, err
	}
	return *seq, nil
}

func (m *gormEngineEventModel) ListAfter(ctx context.Context, seq int64, limit int) ([]*EngineEvent, error) {
	if limit <= 0 {
		limit = 500
	}
	var rows []engineEventRow
	err := m.db.WithContext(ctx).
		Where("id > ?", seq).
		Order("id ASC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*EngineEvent, 0, len(rows))
	for i := range rows {
		out = append(out, engineEventRowToModel(&rows[i]))
	}
	return out, nil
}

func (m *gormEngineEventModel) FindUnresolvedMatches(ctx context.Context, limit int) ([]*EngineEvent, error) {
	if limit <= 0 {
		limit = 1000
	}
	var rows []engineEventRow
	err := m.db.WithContext(ctx).Raw(`
SELECT e.*
FROM engine_events e
WHERE e.event_type = ?
  AND e.match_id <> ''
  AND NOT EXISTS (
      SELECT 1 FROM engine_events done
      WHERE done.match_id = e.match_id
        AND done.event_type IN (?, ?)
  )
ORDER BY e.id ASC
LIMIT ?`,
		EngineEventMatchEnqueued, EngineEventMatchSettled, EngineEventMatchRollback, limit,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*EngineEvent, 0, len(rows))
	for i := range rows {
		out = append(out, engineEventRowToModel(&rows[i]))
	}
	return out, nil
}
