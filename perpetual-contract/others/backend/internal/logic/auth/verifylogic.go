package auth

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"metanode/internal/config"
	"metanode/internal/repo"
	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang-jwt/jwt/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type VerifyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewVerifyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VerifyLogic {
	return &VerifyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func jwtExpireHours(cfg config.AuthConfig) int {
	h := cfg.JwtExpireHours
	if h <= 0 {
		return 168
	}
	return h
}

func verifyPersonalSign(wallet common.Address, message string, sigHex string) bool {
	sigHex = strings.TrimSpace(sigHex)
	hexBody := sigHex
	if strings.HasPrefix(strings.ToLower(sigHex), "0x") {
		hexBody = sigHex[2:]
	}
	sig, err := hex.DecodeString(hexBody)
	if err != nil || len(sig) != 65 {
		return false
	}
	if sig[64] >= 27 {
		sig[64] -= 27
	}
	hash := accounts.TextHash([]byte(message))
	pub, err := crypto.SigToPub(hash, sig)
	if err != nil {
		return false
	}
	recovered := crypto.PubkeyToAddress(*pub)
	return strings.EqualFold(recovered.Hex(), wallet.Hex())
}

func (l *VerifyLogic) AuthVerify(req *types.AuthVerifyReq) (*types.AuthVerifyResp, error) {
	sec := strings.TrimSpace(l.svcCtx.Config.Auth.JwtSecret)
	if sec == "" {
		return &types.AuthVerifyResp{Code: 503, Message: "Auth.JwtSecret 未配置"}, nil
	}

	raw := strings.TrimSpace(req.Address)
	if raw == "" || !common.IsHexAddress(raw) {
		return &types.AuthVerifyResp{Code: 400, Message: "invalid address"}, nil
	}
	wallet := common.HexToAddress(raw)
	addrLower := strings.ToLower(wallet.Hex())

	key := redisNonceKeyPrefix + addrLower
	storedNonce, err := l.svcCtx.Redis.GetCtx(l.ctx, key)
	if err != nil || strings.TrimSpace(storedNonce) == "" {
		return &types.AuthVerifyResp{Code: 401, Message: "nonce expired or missing, request /auth/nonce again"}, nil
	}

	expected := BuildWalletLoginMessage(wallet.Hex(), strings.TrimSpace(storedNonce))
	if strings.TrimSpace(req.Message) != expected {
		return &types.AuthVerifyResp{Code: 400, Message: "message mismatch"}, nil
	}

	if !verifyPersonalSign(wallet, req.Message, req.Signature) {
		return &types.AuthVerifyResp{Code: 401, Message: "invalid signature"}, nil
	}

	if _, err := l.svcCtx.Redis.DelCtx(l.ctx, key); err != nil {
		l.Infof("auth verify del nonce key: %v", err)
	}

	var userID string
	if l.svcCtx.PG != nil {
		id, err := repo.UpsertWalletUserOnLogin(l.ctx, l.svcCtx.PG, addrLower)
		if err != nil {
			return &types.AuthVerifyResp{Code: 500, Message: fmt.Sprintf("persist user: %v", err)}, nil
		}
		userID = id
	}

	hours := jwtExpireHours(l.svcCtx.Config.Auth)
	now := time.Now()
	exp := now.Add(time.Duration(hours) * time.Hour)

	claims := jwt.MapClaims{
		"sub": addrLower,
		"iat": now.Unix(),
		"exp": exp.Unix(),
	}
	if userID != "" {
		claims["uid"] = userID
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(sec))
	if err != nil {
		return &types.AuthVerifyResp{Code: 500, Message: fmt.Sprintf("jwt: %v", err)}, nil
	}

	return &types.AuthVerifyResp{
		Code:      0,
		Message:   "ok",
		Token:     tokenStr,
		ExpiresAt: exp.Unix(),
		UserId:    userID,
		Wallet:    addrLower,
	}, nil
}
