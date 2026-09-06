package repo

import (
	"context"
	"errors"
	"strings"
	"time"

	"metanode/internal/model"

	"gorm.io/gorm"
)

// UpsertWalletUserOnLogin 首次登录插入；再次登录更新 last_login_at（及 updated_at）。
func UpsertWalletUserOnLogin(ctx context.Context, db *gorm.DB, walletLower string) (string, error) {
	walletLower = strings.ToLower(strings.TrimSpace(walletLower))
	if walletLower == "" {
		return "", errors.New("empty wallet address")
	}

	var row model.WalletUser
	err := db.WithContext(ctx).Where("wallet_address = ?", walletLower).First(&row).Error
	now := time.Now().UTC()

	if err == nil {
		row.LastLoginAt = now
		if err := db.WithContext(ctx).Save(&row).Error; err != nil {
			return "", err
		}
		return row.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	row = model.WalletUser{
		WalletAddress: walletLower,
		LastLoginAt:   now,
	}
	if err := db.WithContext(ctx).Create(&row).Error; err != nil {
		return "", err
	}
	return row.ID, nil
}
