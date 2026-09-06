package engine

import (
	"context"
	"time"

	"metanode/internal/config"
	"metanode/internal/model"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// Liquidator 清算机器人
type Liquidator struct {
	config           config.LiquidatorConfig
	ethConfig        config.EthereumConfig
	positionModel    model.PositionModel
	liquidationModel model.LiquidationModel

	stopCh chan struct{}
}

// NewLiquidator 创建清算机器人
func NewLiquidator(cfg config.LiquidatorConfig, ethCfg config.EthereumConfig, db sqlx.SqlConn) *Liquidator {
	return &Liquidator{
		config:           cfg,
		ethConfig:        ethCfg,
		positionModel:    model.NewPositionModel(db),
		liquidationModel: model.NewLiquidationModel(db),
		stopCh:           make(chan struct{}),
	}
}

// Start 启动清算机器人
func (l *Liquidator) Start() {
	if !l.config.Enabled {
		logx.Info("Liquidator is disabled")
		return
	}

	logx.Info("Liquidator starting...")
	go l.checkLoop()
}

// Stop 停止清算机器人
func (l *Liquidator) Stop() {
	close(l.stopCh)
}

// checkLoop 检查循环
func (l *Liquidator) checkLoop() {
	ticker := time.NewTicker(time.Duration(l.config.CheckInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-l.stopCh:
			return
		case <-ticker.C:
			l.checkAllPositions()
		}
	}
}

// checkAllPositions 检查所有仓位
func (l *Liquidator) checkAllPositions() {
	ctx := context.Background()

	// TODO: 从链上获取所有有仓位的交易者
	// traders := l.getAllTraders()

	// 模拟交易者列表
	traders := []string{
		// 从数据库查询有仓位的交易者
	}

	for _, trader := range traders {
		l.checkTraderPosition(ctx, trader)
	}
}

// checkTraderPosition 检查交易者仓位
func (l *Liquidator) checkTraderPosition(ctx context.Context, trader string) {
	// 1. 查询链上安全状态
	isSafe := l.checkSafeOnChain(trader)
	if isSafe {
		return
	}

	logx.Infof("Trader %s is unsafe, preparing liquidation...", trader)

	// 2. 获取仓位信息
	positions, err := l.positionModel.FindByTrader(ctx, trader)
	if err != nil {
		logx.Error("Failed to get positions:", err)
		return
	}

	// 3. 执行清算
	for _, position := range positions {
		l.executeLiquidation(ctx, trader, position)
	}
}

// checkSafeOnChain 检查链上安全状态
func (l *Liquidator) checkSafeOnChain(trader string) bool {
	// TODO: 调用 MetaNodeDealer.isSafe(trader)
	// client, _ := ethclient.Dial(l.ethConfig.RpcUrl)
	// dealer, _ := contracts.NewMetaNodeDealer(common.HexToAddress(l.ethConfig.DealerAddress), client)
	// safe, _ := dealer.IsSafe(nil, common.HexToAddress(trader))
	// return safe

	return true // 占位
}

// executeLiquidation 执行清算
func (l *Liquidator) executeLiquidation(ctx context.Context, trader string, position *model.Position) {
	logx.Infof("Executing liquidation for trader %s, perp %s", trader, position.Perp)

	// TODO: 实现链上清算
	// 1. 计算清算数量和价格
	// 2. 调用 perpetual.liquidate(liquidator, trader, paperAmount, expectCredit)
	// 3. 等待交易确认
	// 4. 记录清算记录

	/*
		// 获取清算人私钥
		privateKey, _ := crypto.HexToECDSA(l.ethConfig.PrivateKey)
		auth, _ := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(l.ethConfig.ChainId))

		// 设置 gas 参数
		auth.GasLimit = uint64(l.ethConfig.MaxGasLimit)

		// 执行清算
		perp, _ := contracts.NewPerpetual(common.HexToAddress(position.Perp), client)
		tx, _ := perp.Liquidate(auth,
			common.HexToAddress(l.ethConfig.LiquidatorAddress),
			common.HexToAddress(trader),
			paperAmount,
			expectCredit,
		)

		// 等待确认
		receipt, _ := bind.WaitMined(ctx, client, tx)

		// 保存记录
		l.liquidationModel.Insert(ctx, &model.Liquidation{
			TxHash:           tx.Hash().Hex(),
			Perp:             position.Perp,
			Liquidator:       l.ethConfig.LiquidatorAddress,
			LiquidatedTrader: trader,
			// ...
		})
	*/
}
