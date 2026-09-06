package engine

// FundingRateKeeper：按 SettleInterval 触发，用标记价与指数价差计算 delta，调用 Dealer.updateFundingRate。
// 链上角色：私钥地址须为 fundingRateKeeper。

import (
	"context"
	"math/big"
	"strings"
	"time"

	"metanode/internal/chain"
	"metanode/internal/config"
	"metanode/internal/model"

	"github.com/ethereum/go-ethereum/common"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// FundingRateKeeper 资金费率结算服务（按配置周期调用 Dealer.updateFundingRate）。
type FundingRateKeeper struct {
	config       config.FundingRateConfig
	markets      []config.MarketConfig
	fundingModel model.FundingRateModel
	chain        *chain.Client

	stopCh chan struct{}
}

// NewFundingRateKeeper 创建资金费率结算服务
func NewFundingRateKeeper(
	cfg config.FundingRateConfig,
	markets []config.MarketConfig,
	db sqlx.SqlConn,
	ch *chain.Client,
) *FundingRateKeeper {
	return &FundingRateKeeper{
		config:       cfg,
		markets:      markets,
		fundingModel: model.NewFundingRateModel(db),
		chain:        ch,
		stopCh:       make(chan struct{}),
	}
}

// Start 启动资金费率服务
func (k *FundingRateKeeper) Start() {
	logx.Info("FundingRateKeeper starting...")
	go k.settleLoop()
}

// Stop 停止服务
func (k *FundingRateKeeper) Stop() {
	close(k.stopCh)
}

func (k *FundingRateKeeper) settleLoop() {
	interval := time.Duration(k.config.SettleInterval) * time.Second
	if interval <= 0 {
		interval = 8 * time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-k.stopCh:
			return
		case <-ticker.C:
			k.settleFundingRate()
		}
	}
}

func (k *FundingRateKeeper) settleFundingRate() {
	// 读取各市场旧累计指数，按溢价算出本次 delta，再 newRate=old+delta 批量上链。
	ctx := context.Background()
	if k.chain == nil {
		logx.Info("FundingRateKeeper: chain client nil, skip")
		return
	}

	type row struct {
		market  config.MarketConfig
		perp    common.Address
		newRate *big.Int
		mark    *big.Int
		idx     *big.Int
	}
	var rows []row

	for _, market := range k.markets {
		perp := common.HexToAddress(market.Address)
		old, err := k.chain.GetFundingRate(ctx, perp)
		if err != nil {
			logx.Errorf("getFundingRate %s: %v", market.Name, err)
			continue
		}
		mark, err := k.chain.GetMarkPrice(ctx, perp)
		if err != nil {
			logx.Errorf("getMarkPrice %s: %v", market.Name, err)
			mark = big.NewInt(0)
		}
		idx := k.indexPrice(ctx, market)
		delta := k.fundingDelta(mark, idx)
		newRate := new(big.Int).Add(old, delta)
		rows = append(rows, row{market, perp, newRate, mark, idx})
	}

	if len(rows) == 0 {
		return
	}

	perps := make([]common.Address, len(rows))
	rates := make([]*big.Int, len(rows))
	for i := range rows {
		perps[i] = rows[i].perp
		rates[i] = rows[i].newRate
	}

	tx, err := k.chain.SubmitFundingRateUpdate(ctx, perps, rates)
	if err != nil {
		logx.Errorf("updateFundingRate tx failed: %v (需保证私钥对应链上 fundingRateKeeper)", err)
		return
	}
	logx.Infof("Funding rate update tx: %s", tx.Hash().Hex())

	for _, r := range rows {
		_, _ = k.fundingModel.Insert(ctx, &model.FundingRate{
			Perp:       r.market.Address,
			Rate:       r.newRate.String(),
			MarkPrice:  r.mark.String(),
			IndexPrice: r.idx.String(),
			SettleTime: time.Now(),
		})
		logx.Infof("Funding settled %s newCumulative=%s", r.market.Name, r.newRate.String())
	}
}

func (k *FundingRateKeeper) indexPrice(ctx context.Context, market config.MarketConfig) *big.Int {
	feed := k.chain.IndexPriceForMarket(market.Name)
	if feed != (common.Address{}) {
		px, err := k.chain.ChainlinkAnswer8(ctx, feed)
		if err == nil {
			return px
		}
		logx.Errorf("chainlink %s: %v", market.Name, err)
	}
	mark, err := k.chain.GetMarkPrice(ctx, common.HexToAddress(market.Address))
	if err != nil {
		return big.NewInt(0)
	}
	return mark
}

// fundingDelta: 溢价 (mark-index)/index 按 8h 周期缩放为累计指数步长（1e18 精度下与合约整数运算一致）。
func (k *FundingRateKeeper) fundingDelta(mark, index *big.Int) *big.Int {
	if index == nil || index.Sign() == 0 || mark == nil {
		return big.NewInt(0)
	}
	premium := new(big.Int).Sub(mark, index)
	premium.Mul(premium, big.NewInt(1e18))
	premium.Div(premium, index)
	// 单次结算假设为日系数的 1/3（约 8h）；可按业务再调
	step := new(big.Int).Div(premium, big.NewInt(3))
	maxStep := parseMaxRateStep(k.config.MaxRate)
	if maxStep != nil && step.Cmp(maxStep) > 0 {
		step.Set(maxStep)
	}
	if maxStep != nil {
		negMax := new(big.Int).Neg(maxStep)
		if step.Cmp(negMax) < 0 {
			step.Set(negMax)
		}
	}
	return step
}

func parseMaxRateStep(s string) *big.Int {
	z := new(big.Int)
	if _, ok := z.SetString(strings.TrimSpace(s), 10); !ok {
		return nil
	}
	if z.Sign() <= 0 {
		return nil
	}
	return z
}
