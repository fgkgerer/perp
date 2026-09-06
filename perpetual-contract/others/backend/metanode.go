package main

import (
	"context"
	"flag"
	"fmt"

	"metanode/internal/chain"
	"metanode/internal/config"
	"metanode/internal/engine"
	"metanode/internal/handler"
	"metanode/internal/handler/market"
	"metanode/internal/listener"
	"metanode/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/metanode.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 允许任意来源跨域（Access-Control-Allow-Origin: *）
	server := rest.MustNewServer(c.RestConf, rest.WithCors())
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	if ctx.PG != nil {
		if sqlDB, err := ctx.PG.DB(); err == nil {
			defer sqlDB.Close()
		}
	}

	// 链客户端：失败时 Chain 为 nil，仅 HTTP 可用，撮合/资金费不落链。
	ch, err := chain.NewClient(c.Ethereum, c.Markets, c.Chainlink)
	if err != nil {
		fmt.Println("chain client:", err, "(链上查询/结算将不可用，请检查 RpcUrl、DealerAddress)")
	} else {
		ctx.Chain = ch
		defer ch.Close()
		if err := config.ValidateSepoliaMarkets(c.Ethereum.ChainId, c.Markets); err != nil {
			fmt.Println("markets config:", err)
		} else {
			fmt.Println("markets config: ok (BTC-PERP, ETH-PERP Sepolia addresses)")
		}
		if err := ch.ValidateOrderSender(context.Background()); err != nil {
			fmt.Println("validOrderSender:", err)
		} else {
			fmt.Println("validOrderSender: ok", ch.From().Hex())
		}
	}

	var treasuryWatcher *listener.TreasuryDepositWatcher
	if ctx.Chain != nil {
		treasuryWatcher = listener.NewTreasuryDepositWatcher(ctx)
		treasuryWatcher.Start()
		defer treasuryWatcher.Stop()
	}

	var indexFeed listener.IndexPriceProvider
	if c.SpotIndex.Enabled {
		composite := listener.NewCompositeSpotIndex(ctx)
		composite.Start()
		indexFeed = composite
		defer composite.Stop()
		w := listener.SpotWeights{c.SpotIndex.CoinbaseWeight, c.SpotIndex.OkxWeight, c.SpotIndex.BinanceWeight}.Normalized()
		logx.Infof("spot index: composite Coinbase:OKX:Binance weights %d:%d:%d", w.Coinbase, w.Okx, w.Binance)
	} else {
		coinbaseFeed := listener.NewCoinbaseFeed(ctx)
		coinbaseFeed.Start()
		indexFeed = coinbaseFeed
		defer coinbaseFeed.Stop()
	}
	ctx.IndexPrice = indexFeed

	// 撮合引擎与 ServiceContext 互相引用，便于下单 API 入簿。
	matchEngine := engine.NewMatchEngine(c.MatchEngine, ctx.OrderModel, ctx.TradeModel, ctx.EngineEventModel, ctx.Chain, ctx.Redis)
	ctx.MatchEngine = matchEngine
	ctx.OrderBook = matchEngine

	handler.RegisterHandlers(server, ctx)
	market.RegisterTickerWebSocket(server, ctx)

	perpAddrs := make([]string, 0, len(c.Markets))
	for _, m := range c.Markets {
		perpAddrs = append(perpAddrs, m.Address)
	}
	matchEngine.Start(perpAddrs)
	defer matchEngine.Stop()

	// 启动清算机器人
	liquidator := engine.NewLiquidator(c.Liquidator, c.Ethereum, ctx.DB)
	liquidator.Start()
	defer liquidator.Stop()

	// 启动资金费率服务
	fundingKeeper := engine.NewFundingRateKeeper(c.FundingRate, c.Markets, ctx.DB, ctx.Chain, indexFeed)
	fundingKeeper.Start()
	defer fundingKeeper.Stop()

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
