package config

import (
	"github.com/zeromicro/go-zero/rest"
)

// Config 应用配置
type Config struct {
	rest.RestConf

	// Redis 配置
	Redis RedisConfig

	// 以太坊配置
	Ethereum EthereumConfig

	// 市场配置
	Markets []MarketConfig

	// 撮合引擎配置
	MatchEngine MatchEngineConfig

	// 资金费率配置
	FundingRate FundingRateConfig

	// 清算机器人配置
	Liquidator LiquidatorConfig

	// Chainlink 预言机配置
	Chainlink ChainlinkConfig

	// Supabase Postgres（GORM DataSource；与 YAML Supabase 段对齐）
	Supabase SupabaseConfig

	// USDC 转收款地址 address_a 的扫链入账（充值记录 + 链下余额）
	TreasuryDeposit TreasuryDepositConfig

	// 钱包 EIP-191 签名登录（JWT）
	Auth AuthConfig `json:",optional"`

	// Coinbase 现货 ticker WS（指数价推送）
	Coinbase CoinbaseConfig `json:",optional"`

	// SpotIndex 三所加权现货指数（Coinbase:OKX:Binance 默认 4:3:3）
	SpotIndex SpotIndexConfig `json:",optional"`
}

// AuthConfig 与 etc/metanode.yaml 中 Auth 段对应。
type AuthConfig struct {
	JwtSecret       string `json:",optional"`
	JwtExpireHours  int    `json:",optional"` // 默认 168
	NonceTTLMinutes int    `json:",optional"` // 默认 15
}

// SupabaseConfig Postgres（服务端可选）
type SupabaseConfig struct {
	DataSource   string `json:",optional"`
	MaxOpenConns int    `json:",optional"`
	MaxIdleConns int    `json:",optional"`
}

// TreasuryDepositConfig 监听 USDC Transfer(to = UsdcTreasuryAddress)，记入 deposits / ledger_balances。
type TreasuryDepositConfig struct {
	Enabled               bool   `json:",optional"`
	PollIntervalSeconds   int    `json:",optional"` // 默认 12
	Confirmations         uint64 `json:",optional"` // 默认 1，收据区块距今确认数
	InitialLookbackBlocks uint64 `json:",optional"` // Redis 无游标时从 head-lookback 开始；默认 2000
}

// MySQLConfig MySQL 配置
type MySQLConfig struct {
	DataSource string
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host string
	Type string
	Pass string
}

// EthereumConfig 以太坊配置
type EthereumConfig struct {
	RpcUrl        string
	ChainId       int64
	DealerAddress string
	// UsdcAddress ERC20 USDC 合约地址（扫 Transfer）
	UsdcAddress string `json:",optional"`
	// UsdcTreasuryAddress 平台收款地址 address_a，用户往该地址转 USDC 即充值入账。
	UsdcTreasuryAddress string `json:",optional"`
	PrivateKey          string
	GasPriceMultiplier  float64
	MaxGasLimit         int64
}

// MarketConfig 市场配置
type MarketConfig struct {
	Address         string
	Name            string
	PriceSource     string
	CoinbaseProduct string `json:",optional"` // 如 BTC-USD
	BinanceSymbol   string `json:",optional"` // 如 BTCUSDT
	OkxInstId       string `json:",optional"` // 如 BTC-USDT
	// 链上 getRiskParams 不可用时的展示兜底（与 DeploySepolia 默认一致）
	MaxLeverage          string `json:",optional"`
	InitialMarginPct     string `json:",optional"`
	MaintenanceMarginPct string `json:",optional"`
}

// SpotIndexConfig 多交易所现货指数（加权后供 API / 资金费）。
type SpotIndexConfig struct {
	Enabled        bool   `json:",optional"`
	CoinbaseWeight int    `json:",optional"` // 默认 4
	OkxWeight      int    `json:",optional"` // 默认 3
	BinanceWeight  int    `json:",optional"` // 默认 3
	BinanceWsURL   string `json:",optional"` // 空则自动拼 combined stream
	OkxWsURL       string `json:",optional"` // 默认 wss://ws.okx.com:8443/ws/v5/public
}

// CoinbaseConfig Coinbase Exchange WS ticker
type CoinbaseConfig struct {
	Enabled      bool   `json:",optional"`
	WsURL        string `json:",optional"` // 默认 wss://ws-feed.exchange.coinbase.com
	ThrottleMs   int    `json:",optional"` // 写库节流，默认 500ms
	ReconnectSec int    `json:",optional"` // 断线重连间隔，默认 5
}

// MatchEngineConfig 撮合引擎配置
type MatchEngineConfig struct {
	MatchInterval        int
	BatchSize            int
	MaxPendingOrders     int
	LeaderLockKey        string `json:",optional"`
	LeaderLockTTLSeconds int    `json:",optional"`
	LeaderRenewSeconds   int    `json:",optional"`
	EventPollIntervalMs  int    `json:",optional"`
}

// FundingRateConfig 资金费率配置
type FundingRateConfig struct {
	SettleInterval int
	MaxRate        string
}

// LiquidatorConfig 清算机器人配置
type LiquidatorConfig struct {
	CheckInterval int
	Enabled       bool
}

// ChainlinkConfig Chainlink 预言机配置
type ChainlinkConfig struct {
	BTC_USD string
	ETH_USD string
}
