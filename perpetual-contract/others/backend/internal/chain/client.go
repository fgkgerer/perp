// Package chain 封装与 MetaNodeDealer / Perpetual 的链上交互：
// 只读调用走 Dealer；成交走 Perp.trade；资金费率更新走 Dealer.updateFundingRate。
// 订单签名域必须与链上 MetaNodeStorage.domainSeparator 一致（MetaNode / v1 / chainId / Dealer 地址）。
package chain

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"metanode/internal/config"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Client 封装 Dealer / Perpetual 的 RPC 与（可选）发送交易能力。
type Client struct {
	cfg           config.EthereumConfig
	markets       []config.MarketConfig
	chainlink     config.ChainlinkConfig
	eth           *ethclient.Client
	dealer        common.Address
	dealerABI     abi.ABI
	perpABI       abi.ABI
	aggABI        abi.ABI
	signerKey     *ecdsa.PrivateKey
	chainID       *big.Int
	fromAddr      common.Address
	maxGas        uint64
	gasMultiplier float64
}

// NewClient 连接 RPC 并解析 ABI。若配置 PrivateKey，则 From() 可用于发交易（撮合、资金费等）。
func NewClient(cfg config.EthereumConfig, markets []config.MarketConfig, chainlink config.ChainlinkConfig) (*Client, error) {
	ethc, err := ethclient.Dial(cfg.RpcUrl)
	if err != nil {
		return nil, fmt.Errorf("eth dial: %w", err)
	}
	dABI, err := abi.JSON(strings.NewReader(dealerABIStr))
	if err != nil {
		return nil, err
	}
	pABI, err := abi.JSON(strings.NewReader(perpetualABIStr))
	if err != nil {
		return nil, err
	}
	aABI, err := abi.JSON(strings.NewReader(aggregatorV3ABIStr))
	if err != nil {
		return nil, err
	}
	c := &Client{
		cfg:           cfg,
		markets:       markets,
		chainlink:     chainlink,
		eth:           ethc,
		dealer:        common.HexToAddress(cfg.DealerAddress),
		dealerABI:     dABI,
		perpABI:       pABI,
		aggABI:        aABI,
		chainID:       big.NewInt(cfg.ChainId),
		maxGas:        uint64(cfg.MaxGasLimit),
		gasMultiplier: cfg.GasPriceMultiplier,
	}
	if cfg.GasPriceMultiplier <= 0 {
		c.gasMultiplier = 1.0
	}
	if cfg.PrivateKey != "" {
		key, err := crypto.HexToECDSA(strings.TrimPrefix(cfg.PrivateKey, "0x"))
		if err != nil {
			return nil, fmt.Errorf("private key: %w", err)
		}
		c.signerKey = key
		c.fromAddr = crypto.PubkeyToAddress(key.PublicKey)
	}
	return c, nil
}

func (c *Client) From() common.Address { return c.fromAddr }

func (c *Client) Dealer() common.Address { return c.dealer }

func (c *Client) Close() { c.eth.Close() }

// RPC 返回底层 RPC 客户端（例如 WaitMined）。
func (c *Client) RPC() *ethclient.Client { return c.eth }

func (c *Client) Markets() []config.MarketConfig { return c.markets }

// IsOrderSenderValid 查询 Dealer 上 orderSender 是否为 validOrderSender。
func (c *Client) IsOrderSenderValid(ctx context.Context, sender common.Address) (bool, error) {
	data, err := c.dealerABI.Pack("isOrderSenderValid", sender)
	if err != nil {
		return false, err
	}
	out, err := c.callDealer(ctx, data)
	if err != nil {
		return false, err
	}
	res, err := c.dealerABI.Unpack("isOrderSenderValid", out)
	if err != nil {
		return false, err
	}
	return res[0].(bool), nil
}

// ValidateOrderSender 启动时校验配置私钥地址是否为链上 validOrderSender。
func (c *Client) ValidateOrderSender(ctx context.Context) error {
	if c.signerKey == nil {
		return fmt.Errorf("Ethereum.PrivateKey 未配置，无法提交链上成交")
	}
	valid, err := c.IsOrderSenderValid(ctx, c.fromAddr)
	if err != nil {
		return fmt.Errorf("查询 validOrderSender 失败: %w", err)
	}
	if !valid {
		return fmt.Errorf("PrivateKey 对应地址 %s 不是 Dealer.validOrderSender，撮合上链将 revert", c.fromAddr.Hex())
	}
	return nil
}

func (c *Client) callDealer(ctx context.Context, data []byte) ([]byte, error) {
	msg := ethereum.CallMsg{To: &c.dealer, Data: data}
	return c.eth.CallContract(ctx, msg, nil)
}

func (c *Client) callPerp(ctx context.Context, perp common.Address, data []byte) ([]byte, error) {
	msg := ethereum.CallMsg{To: &perp, Data: data}
	return c.eth.CallContract(ctx, msg, nil)
}

// GetMarkPrice 调用 Dealer.getMarkPrice(perp)，一般为 1e18 精度。
func (c *Client) GetMarkPrice(ctx context.Context, perp common.Address) (*big.Int, error) {
	data, err := c.dealerABI.Pack("getMarkPrice", perp)
	if err != nil {
		return nil, err
	}
	out, err := c.callDealer(ctx, data)
	if err != nil {
		return nil, err
	}
	res, err := c.dealerABI.Unpack("getMarkPrice", out)
	if err != nil {
		return nil, err
	}
	return res[0].(*big.Int), nil
}

// GetFundingRate 返回 Perp 侧累计资金指数（合约中为累计值，单次结算看链上变化量）。
func (c *Client) GetFundingRate(ctx context.Context, perp common.Address) (*big.Int, error) {
	data, err := c.dealerABI.Pack("getFundingRate", perp)
	if err != nil {
		return nil, err
	}
	out, err := c.callDealer(ctx, data)
	if err != nil {
		return nil, err
	}
	res, err := c.dealerABI.Unpack("getFundingRate", out)
	if err != nil {
		return nil, err
	}
	return res[0].(*big.Int), nil
}

// PerpRiskParams 链上 Types.RiskParams（1e18 精度比率 + 元数据）。
type PerpRiskParams struct {
	InitialMarginRatio   *big.Int
	LiquidationThreshold *big.Int
	LiquidationPriceOff  *big.Int
	InsuranceFeeRate     *big.Int
	MarkPriceSource      common.Address
	Name                 string
	IsRegistered         bool
}

// GetRiskParams 读取永续合约风险参数（杠杆 / 清算阈值等）。
func (c *Client) GetRiskParams(ctx context.Context, perp common.Address) (*PerpRiskParams, error) {
	data, err := c.dealerABI.Pack("getRiskParams", perp)
	if err != nil {
		return nil, err
	}
	raw, err := c.callDealer(ctx, data)
	if err != nil {
		return nil, err
	}
	res, err := c.dealerABI.Unpack("getRiskParams", raw)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, fmt.Errorf("getRiskParams: empty result")
	}
	tuple, ok := res[0].(struct {
		InitialMarginRatio   *big.Int       `json:"initialMarginRatio"`
		LiquidationThreshold *big.Int       `json:"liquidationThreshold"`
		LiquidationPriceOff  *big.Int       `json:"liquidationPriceOff"`
		InsuranceFeeRate     *big.Int       `json:"insuranceFeeRate"`
		MarkPriceSource      common.Address `json:"markPriceSource"`
		Name                 string         `json:"name"`
		IsRegistered         bool           `json:"isRegistered"`
	})
	if !ok {
		return nil, fmt.Errorf("getRiskParams: unexpected type %T", res[0])
	}
	return &PerpRiskParams{
		InitialMarginRatio:   tuple.InitialMarginRatio,
		LiquidationThreshold: tuple.LiquidationThreshold,
		LiquidationPriceOff:  tuple.LiquidationPriceOff,
		InsuranceFeeRate:     tuple.InsuranceFeeRate,
		MarkPriceSource:      tuple.MarkPriceSource,
		Name:                 tuple.Name,
		IsRegistered:         tuple.IsRegistered,
	}, nil
}

// GetCreditOf 对应 IDealer.getCreditOf：主/次资产 credit、待提现与时间锁时间戳。
func (c *Client) GetCreditOf(ctx context.Context, trader common.Address) (
	primaryCredit *big.Int,
	secondaryCredit *big.Int,
	pendingPrimary *big.Int,
	pendingSecondary *big.Int,
	execTs *big.Int,
	err error,
) {
	data, err := c.dealerABI.Pack("getCreditOf", trader)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	out, err := c.callDealer(ctx, data)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	res, err := c.dealerABI.Unpack("getCreditOf", out)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if len(res) != 5 {
		return nil, nil, nil, nil, nil, fmt.Errorf("getCreditOf: unexpected arity %d", len(res))
	}
	return res[0].(*big.Int), res[1].(*big.Int), res[2].(*big.Int), res[3].(*big.Int), res[4].(*big.Int), nil
}

// GetTraderRisk 返回净值、敞口、初始/维持保证金（与链上风控计算一致）。
func (c *Client) GetTraderRisk(ctx context.Context, trader common.Address) (
	netValue, exposure, initialMargin, maintenanceMargin *big.Int, err error,
) {
	data, err := c.dealerABI.Pack("getTraderRisk", trader)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	out, err := c.callDealer(ctx, data)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	res, err := c.dealerABI.Unpack("getTraderRisk", out)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return res[0].(*big.Int), res[1].(*big.Int), res[2].(*big.Int), res[3].(*big.Int), nil
}

// GetPositions 列出该用户仍持仓的 Perp 合约地址列表。
func (c *Client) GetPositions(ctx context.Context, trader common.Address) ([]common.Address, error) {
	data, err := c.dealerABI.Pack("getPositions", trader)
	if err != nil {
		return nil, err
	}
	out, err := c.callDealer(ctx, data)
	if err != nil {
		return nil, err
	}
	res, err := c.dealerABI.Unpack("getPositions", out)
	if err != nil {
		return nil, err
	}
	addrs, ok := res[0].([]common.Address)
	if !ok {
		return nil, fmt.Errorf("getPositions: unexpected type %T", res[0])
	}
	return addrs, nil
}

// GetLiquidationPrice 参考清算价（链上近似值，注释见 Dealer）。
func (c *Client) GetLiquidationPrice(ctx context.Context, trader, perp common.Address) (*big.Int, error) {
	data, err := c.dealerABI.Pack("getLiquidationPrice", trader, perp)
	if err != nil {
		return nil, err
	}
	out, err := c.callDealer(ctx, data)
	if err != nil {
		return nil, err
	}
	res, err := c.dealerABI.Unpack("getLiquidationPrice", out)
	if err != nil {
		return nil, err
	}
	return res[0].(*big.Int), nil
}

// IsSafe 是否满足维持保证金（不会被清算）。
func (c *Client) IsSafe(ctx context.Context, trader common.Address) (bool, error) {
	data, err := c.dealerABI.Pack("isSafe", trader)
	if err != nil {
		return false, err
	}
	out, err := c.callDealer(ctx, data)
	if err != nil {
		return false, err
	}
	res, err := c.dealerABI.Unpack("isSafe", out)
	if err != nil {
		return false, err
	}
	return res[0].(bool), nil
}

// BalanceOf 调用 Perpetual.balanceOf：paper 仓位、credit 含资金费累计调整后的值。
func (c *Client) BalanceOf(ctx context.Context, perp, trader common.Address) (paper, credit *big.Int, err error) {
	data, err := c.perpABI.Pack("balanceOf", trader)
	if err != nil {
		return nil, nil, err
	}
	out, err := c.callPerp(ctx, perp, data)
	if err != nil {
		return nil, nil, err
	}
	res, err := c.perpABI.Unpack("balanceOf", out)
	if err != nil {
		return nil, nil, err
	}
	return res[0].(*big.Int), res[1].(*big.Int), nil
}

// ChainlinkAnswer8 返回 Chainlink feed 的 answer（通常为 8 位小数），放大到 1e18 精度以与标记价格口径对齐。
func (c *Client) ChainlinkAnswer8(ctx context.Context, feed common.Address) (*big.Int, error) {
	if feed == (common.Address{}) {
		return nil, fmt.Errorf("empty feed address")
	}
	data, err := c.aggABI.Pack("latestRoundData")
	if err != nil {
		return nil, err
	}
	out, err := c.eth.CallContract(ctx, ethereum.CallMsg{To: &feed, Data: data}, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.aggABI.Unpack("latestRoundData", out)
	if err != nil {
		return nil, err
	}
	answer := res[1].(*big.Int)
	// 假设 decimals=8：answer * 1e10 -> 1e18
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(10), nil)
	return new(big.Int).Mul(answer, scale), nil
}

// IndexPriceForMarket 按市场名称解析配置中的 Chainlink 聚合器地址（无则返回零地址）。
func (c *Client) IndexPriceForMarket(name string) common.Address {
	switch name {
	case "BTC-PERP":
		return common.HexToAddress(c.chainlink.BTC_USD)
	case "ETH-PERP":
		return common.HexToAddress(c.chainlink.ETH_USD)
	default:
		return common.Address{}
	}
}

// SubmitPerpTrade 以配置私钥账户调用 perp.trade(tradeData)。该账户须为 Dealer 上登记的 validOrderSender。
func (c *Client) SubmitPerpTrade(ctx context.Context, perp common.Address, tradeData []byte) (*types.Transaction, error) {
	if c.signerKey == nil {
		return nil, fmt.Errorf("no PrivateKey configured for on-chain submit")
	}
	data, err := c.perpABI.Pack("trade", tradeData)
	if err != nil {
		return nil, err
	}
	nonce, err := c.eth.PendingNonceAt(ctx, c.fromAddr)
	if err != nil {
		return nil, err
	}
	gasPrice, err := c.eth.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}
	if c.gasMultiplier > 0 && c.gasMultiplier != 1 {
		f := big.NewFloat(c.gasMultiplier)
		gp, _ := f.Mul(f, new(big.Float).SetInt(gasPrice)).Int(nil)
		if gp.Sign() > 0 {
			gasPrice = gp
		}
	}
	msg := ethereum.CallMsg{From: c.fromAddr, To: &perp, Gas: c.maxGas, GasPrice: gasPrice, Data: data}
	gasLimit, err := c.eth.EstimateGas(ctx, msg)
	if err != nil {
		gasLimit = c.maxGas
	}
	if gasLimit > c.maxGas {
		gasLimit = c.maxGas
	}
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		GasPrice: gasPrice,
		Gas:      gasLimit,
		To:       &perp,
		Value:    big.NewInt(0),
		Data:     data,
	})
	signer := types.LatestSignerForChainID(c.chainID)
	signed, err := types.SignTx(tx, signer, c.signerKey)
	if err != nil {
		return nil, err
	}
	if err := c.eth.SendTransaction(ctx, signed); err != nil {
		return nil, err
	}
	return signed, nil
}

// SubmitFundingRateUpdate 调用 Dealer.updateFundingRate。交易发送者须为链上 fundingRateKeeper。
func (c *Client) SubmitFundingRateUpdate(ctx context.Context, perps []common.Address, rates []*big.Int) (*types.Transaction, error) {
	if c.signerKey == nil {
		return nil, fmt.Errorf("no PrivateKey configured for funding update")
	}
	if len(perps) != len(rates) {
		return nil, fmt.Errorf("perps/rates length mismatch")
	}
	data, err := c.dealerABI.Pack("updateFundingRate", perps, rates)
	if err != nil {
		return nil, err
	}
	nonce, err := c.eth.PendingNonceAt(ctx, c.fromAddr)
	if err != nil {
		return nil, err
	}
	gasPrice, err := c.eth.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}
	if c.gasMultiplier > 0 && c.gasMultiplier != 1 {
		f := big.NewFloat(c.gasMultiplier)
		gp, _ := f.Mul(f, new(big.Float).SetInt(gasPrice)).Int(nil)
		if gp.Sign() > 0 {
			gasPrice = gp
		}
	}
	msg := ethereum.CallMsg{From: c.fromAddr, To: &c.dealer, Gas: c.maxGas, GasPrice: gasPrice, Data: data}
	gasLimit, err := c.eth.EstimateGas(ctx, msg)
	if err != nil {
		gasLimit = 500000
	}
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		GasPrice: gasPrice,
		Gas:      gasLimit,
		To:       &c.dealer,
		Value:    big.NewInt(0),
		Data:     data,
	})
	signer := types.LatestSignerForChainID(c.chainID)
	signed, err := types.SignTx(tx, signer, c.signerKey)
	if err != nil {
		return nil, err
	}
	if err := c.eth.SendTransaction(ctx, signed); err != nil {
		return nil, err
	}
	return signed, nil
}
