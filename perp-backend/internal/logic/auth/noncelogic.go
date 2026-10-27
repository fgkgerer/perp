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
	//这行代码的作用是：把 16 字节的随机二进制数据，转换成 32 个字符的十六进制字符串。
	/**
					buf := make([]byte, 16)          // 16 字节的随机二进制数据
				rand.Read(buf)                    // 填充随机数
				nonce := hex.EncodeToString(buf)  // 转成十六进制字符串
				hex.EncodeToString 的逻辑就是：把每个字节拆成高 4 位和低 4 位，各自映射成一个十六进制字符。
				buf = [0xAB, 0x3F, 0x07, ...]   // 16 个字节
						↓
				nonce = "ab3f07..."              // 32 个十六进制字符
				长度变化：16 字节 → 32 字符（每个字节变成 2 个 hex 字符）。
			为什么要做这个转换？
			buf 是原始二进制数据，里面可能包含不可打印字符（比如 0x00、0xFF），直接传给前端或存到 Redis 都不方便。转成十六进制字符串后：
		全是可打印的 ASCII 字符（0-9, a-f）
		方便 JSON 传输、Redis 存储、日志打印
		后续用 hex.DecodeString() 可以无损还原回原始字节

		在这段代码中的作用
	生成的 nonce 是一个随机字符串，用来防止重放攻击：
	服务端生成随机 nonce，存到 Redis（带过期时间）
	把 nonce 拼进签名消息，发给前端
	前端用钱包签名后，连同 nonce 一起发回来
	服务端验证签名时，检查 Redis 里有没有这个 nonce，用完就删
	这样每个登录请求的签名消息都不一样，即使有人截获了签名，也无法重复使用。

	简单总结：hex.EncodeToString(buf) 就是把随机二进制数据"翻译"成人类可读的十六进制字符串，方便传输和存储。
	**/
	nonce := hex.EncodeToString(buf)
	msg := BuildWalletLoginMessage(addr.Hex(), nonce) //addr.Hex()用于 BuildWalletLoginMessage 构建签名消息，确保消息内容标准化

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
