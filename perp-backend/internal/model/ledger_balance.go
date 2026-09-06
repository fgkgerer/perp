package model

import (
	"context"
	"database/sql"
	"math/big"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// LedgerBalance 链下入账累计（USDC 最小单位字符串，与 deposits.primary_amount 口径一致）。
type LedgerBalance struct {
	Trader         string    `db:"trader"`
	PrimaryBalance string    `db:"primary_balance"`
	UpdatedAt      time.Time `db:"updated_at"`
}

type LedgerBalanceModel interface {
	GetPrimary(ctx context.Context, traderLower string) (*big.Int, error)
	AddPrimary(ctx context.Context, traderLower string, delta *big.Int) error
}

type defaultLedgerBalanceModel struct {
	conn sqlx.SqlConn
}

func NewLedgerBalanceModel(conn sqlx.SqlConn) LedgerBalanceModel {
	return &defaultLedgerBalanceModel{conn: conn}
}

func normalizeTraderKey(trader string) string {
	return strings.ToLower(strings.TrimSpace(trader))
}

func (m *defaultLedgerBalanceModel) GetPrimary(ctx context.Context, traderLower string) (*big.Int, error) {
	key := normalizeTraderKey(traderLower)
	var row LedgerBalance
	err := m.conn.QueryRowCtx(ctx, &row, `SELECT trader, primary_balance, updated_at FROM ledger_balances WHERE trader = ? LIMIT 1`, key)
	if err == sql.ErrNoRows {
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

func (m *defaultLedgerBalanceModel) AddPrimary(ctx context.Context, traderLower string, delta *big.Int) error {
	if delta == nil || delta.Sign() <= 0 {
		return nil
	}
	key := normalizeTraderKey(traderLower)
	cur, err := m.GetPrimary(ctx, key)
	if err != nil {
		return err
	}
	sum := new(big.Int).Add(cur, delta)
	now := time.Now()
	_, err = m.conn.ExecCtx(ctx,
		`INSERT INTO ledger_balances (trader, primary_balance, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT (trader) DO UPDATE SET primary_balance = EXCLUDED.primary_balance, updated_at = EXCLUDED.updated_at`,
		key, sum.String(), now)
	return err
}
