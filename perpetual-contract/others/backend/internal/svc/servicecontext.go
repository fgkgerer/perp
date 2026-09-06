package svc

import (
	"fmt"
	"strings"

	"metanode/internal/chain"
	"metanode/internal/config"
	"metanode/internal/db"
	"metanode/internal/model"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// MatchBookSink 订单入 memory book（由 internal/engine.MatchEngine 实现）。
type MatchBookSink interface {
	AddOrder(order *model.Order) error
	RemoveOrder(orderID string) bool
}

// OrderBookLevel 内存订单簿档位快照。
type OrderBookLevel struct {
	Price  string
	Amount string
}

// OrderBookSource 撮合引擎订单簿快照（由 MatchEngine 实现）。
type OrderBookSource interface {
	SnapshotOrderBook(perp string, limit int) (bids, asks []OrderBookLevel)
	// MidMarkPrice 双边为 mid，单边为最优买一或卖一；空簿 ok=false。
	MidMarkPrice(perp string) (price string, ok bool)
}

// IndexPriceSource 现货指数价（CompositeSpotIndex 4:3:3 或 CoinbaseFeed）。
type IndexPriceSource interface {
	IndexPrice1e6(perp string) (price string, ok bool)
	IndexPriceDisplay(perp string) (price string, ok bool)
}

// ServiceContext 服务上下文
type ServiceContext struct {
	Config config.Config

	// 数据库连接
	DB sqlx.SqlConn

	// PG Supabase / Postgres（GORM）；未配置 DataSource 时为 nil。
	PG *gorm.DB

	// Redis 连接
	Redis *redis.Redis

	// 数据模型
	OrderModel         model.OrderModel
	TradeModel         model.TradeModel
	PositionModel      model.PositionModel
	DepositModel       model.DepositModel
	WithdrawModel      model.WithdrawModel
	FundingRateModel   model.FundingRateModel
	LiquidationModel   model.LiquidationModel
	MarketModel        model.MarketModel
	KlineModel         model.KlineModel
	LedgerBalanceModel model.LedgerBalanceModel
	MarketQuoteModel   model.MarketQuoteModel
	SpotKlineModel     model.SpotKlineModel
	EngineEventModel   model.EngineEventModel

	// Chain 为 nil 时：无 RPC，链上查询接口返回空或零；撮合仍可记库但不会发交易。
	Chain *chain.Client
	// MatchEngine 在 main 中创建并赋值，供 CreateOrder 入簿。
	MatchEngine MatchBookSink
	// OrderBook 在 main 中与 MatchEngine 同一实例，供深度 API 读取。
	OrderBook OrderBookSource
	// IndexPrice Coinbase 指数价（可选）。
	IndexPrice IndexPriceSource
}

// NewServiceContext 创建服务上下文
func NewServiceContext(c config.Config) *ServiceContext {
	// 初始化 Redis 连接
	rds := redis.MustNewRedis(redis.RedisConf{
		Host: c.Redis.Host,
		Type: c.Redis.Type,
		Pass: c.Redis.Pass,
	})

	var pgDB *gorm.DB
	var sqlConn sqlx.SqlConn

	if ds := strings.TrimSpace(c.Supabase.DataSource); ds != "" {
		if !strings.Contains(ds, "sslmode=") {
			if strings.Contains(ds, "?") {
				ds += "&sslmode=require"
			} else {
				ds += "?sslmode=require"
			}
		}
		gdb, err := gorm.Open(postgres.Open(ds), &gorm.Config{})
		if err != nil {
			fmt.Println("supabase postgres (gorm):", err, "(未连接 PG，请检查 Supabase.DataSource)")
		} else {
			sqlDB, errSQL := gdb.DB()
			if errSQL != nil {
				fmt.Println("supabase postgres sql.DB:", errSQL)
			} else {
				if mo := c.Supabase.MaxOpenConns; mo > 0 {
					sqlDB.SetMaxOpenConns(mo)
				}
				if mi := c.Supabase.MaxIdleConns; mi > 0 {
					sqlDB.SetMaxIdleConns(mi)
				}
				pgDB = gdb
				// model 层仍使用 go-zero sqlx；占位符由 sqlx 展开后与 Postgres 兼容。
				sqlConn = sqlx.NewSqlConnFromDB(sqlDB)
				if err := db.MigrateAccountTables(pgDB); err != nil {
					fmt.Println("postgres migrate:", err)
				} else {
					fmt.Println("postgres migrate: ok (users, deposits, ledger_balances, orders, trades, funding_rates, market_quotes, spot_klines)")
				}
			}
		}
	} else {
		fmt.Println("supabase: Supabase.DataSource 为空，链下余额/充值记录不可用")
	}

	svcCtx := &ServiceContext{
		Config: c,
		DB:     sqlConn,
		PG:     pgDB,
		Redis:  rds,
	}

	if sqlConn != nil {
		svcCtx.PositionModel = model.NewPositionModel(sqlConn)
		svcCtx.WithdrawModel = model.NewWithdrawModel(sqlConn)
		svcCtx.FundingRateModel = model.NewFundingRateModel(sqlConn)
		svcCtx.LiquidationModel = model.NewLiquidationModel(sqlConn)
		svcCtx.MarketModel = model.NewMarketModel(sqlConn)
		svcCtx.KlineModel = model.NewKlineModel(sqlConn)
	}
	// 充值 / 链下余额 / 订单 / 成交：Postgres 上必须用 GORM（go-zero sqlx 的 ? 与 pgx 不兼容）
	if pgDB != nil {
		svcCtx.DepositModel = model.NewGormDepositModel(pgDB)
		svcCtx.LedgerBalanceModel = model.NewGormLedgerBalanceModel(pgDB)
		svcCtx.OrderModel = model.NewGormOrderModel(pgDB)
		svcCtx.TradeModel = model.NewGormTradeModel(pgDB)
		svcCtx.MarketQuoteModel = model.NewGormMarketQuoteModel(pgDB)
		svcCtx.SpotKlineModel = model.NewGormSpotKlineModel(pgDB)
		svcCtx.EngineEventModel = model.NewGormEngineEventModel(pgDB)
	}

	return svcCtx
}
