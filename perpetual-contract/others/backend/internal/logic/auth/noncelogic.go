package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"metanode/internal/config"
	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/zeromicro/go-zero/core/logx"
)

const redisNonceKeyPrefix = "metanode:wallet_auth_nonce:"

type NonceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewNonceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NonceLogic {
	return &NonceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func authNonceTTL(cfg config.AuthConfig) int {
	m := cfg.NonceTTLMinutes
	if m <= 0 {
		m = 15
	}
	return m * 60
}

func (l *NonceLogic) AuthNonce(req *types.AuthNonceReq) (*types.AuthNonceResp, error) {
	sec := strings.TrimSpace(l.svcCtx.Config.Auth.JwtSecret)
	if sec == "" {
		return &types.AuthNonceResp{Code: 503, Message: "Auth.JwtSecret 未配置"}, nil
	}

	raw := strings.TrimSpace(req.Address)
	if raw == "" || !common.IsHexAddress(raw) {
		return &types.AuthNonceResp{Code: 400, Message: "invalid address"}, nil
	}

	addr := common.HexToAddress(raw)
	addrLower := strings.ToLower(addr.Hex())

	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return &types.AuthNonceResp{Code: 500, Message: fmt.Sprintf("nonce: %v", err)}, nil
	}
	nonce := hex.EncodeToString(buf)
	msg := BuildWalletLoginMessage(addr.Hex(), nonce)

	key := redisNonceKeyPrefix + addrLower
	ttl := authNonceTTL(l.svcCtx.Config.Auth)
	if err := l.svcCtx.Redis.SetexCtx(l.ctx, key, nonce, ttl); err != nil {
		return &types.AuthNonceResp{Code: 500, Message: fmt.Sprintf("redis: %v", err)}, nil
	}

	return &types.AuthNonceResp{
		Code:          0,
		Message:       "ok",
		Nonce:         nonce,
		MessageToSign: msg,
	}, nil
}
