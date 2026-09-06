package model

import "time"

// WalletUser 对应 public.users（钱包登录），非 Supabase Auth 的 auth.users。
type WalletUser struct {
	ID            string    `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()"`
	WalletAddress string    `gorm:"column:wallet_address;size:66;not null;uniqueIndex"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime"`
	LastLoginAt   time.Time `gorm:"column:last_login_at"`
}

func (WalletUser) TableName() string {
	return "users"
}
