/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

import "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import "forge-std/Test.sol";
import "../../src/MetaNodeDealer.sol";
import "../../src/Perpetual.sol";
import "../../src/libraries/Types.sol";
import "../../src/support/TestERC20.sol";
import "../../src/support/TestMarkPriceSource.sol";
import "../utils/EIP712Test.sol";
import "../utils/Utils.sol";

/**
 * @title Cheats - Foundry 测试作弊码接口
 * @notice 用于访问 Foundry 的 VM 作弊码功能
 */
interface Cheats {
    /// @notice 期望下一个调用 revert
    function expectRevert() external;

    /// @notice 期望下一个调用以特定消息 revert
    function expectRevert(bytes calldata) external;
}

/**
 * @title TradingInit - 交易测试初始化合约
 * @notice 为永续合约交易测试提供基础设置
 * 
 * 测试环境包括：
 * 1. 测试代币：USDC（主资产）、MUSD（次级资产）
 * 2. MetaNodeDealer：核心交易系统
 * 3. 两个永续合约市场：BTC-PERP、ETH-PERP
 * 4. 测试用户和交易者
 * 
 * 风险参数设置：
 * - BTC-PERP: 5% 初始保证金率（20倍杠杆），3% 维持保证金率
 * - ETH-PERP: 10% 初始保证金率（10倍杠杆），5% 维持保证金率
 */
contract TradingInit is Test {
    // 排除在覆盖率报告之外
    function test() public { }

    /// @notice Foundry 作弊码地址
    Cheats internal constant cheats = Cheats(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    // ==================== 测试合约实例 ====================
    
    /// @notice 次级资产代币（MUSD）
    TestERC20 public musd;
    /// @notice 主资产代币（USDC）
    TestERC20 public usdc;
    /// @notice 交易系统核心合约
    MetaNodeDealer public metaNodeDealer;
    /// @notice 测试工具合约
    Utils internal utils;
    /// @notice 永续合约列表 [BTC-PERP, ETH-PERP]
    Perpetual[] internal perpList;
    /// @notice 价格源列表
    TestMarkPriceSource[] internal priceSourceList;

    // ==================== 测试账户 ====================
    
    /// @notice 交易者地址列表
    address[] internal traders;
    /// @notice 保险账户地址
    address public insurance;
    /// @notice 通用测试用户
    address payable[] internal users;
    /// @notice 交易者私钥列表（用于签名）
    uint256[] internal tradersKey;

    /**
     * @notice 初始化测试用户
     * @dev 创建：
     *      - 5 个通用用户（每个有 100 ETH）
     *      - 3 个交易者（用固定私钥生成，便于签名测试）
     */
    function initUsers() public {
        utils = new Utils();
        users = utils.createUsers(5);
        // 第一个用户作为保险账户
        insurance = users[0];
        vm.label(insurance, "insurance");

        // 创建 3 个交易者
        traders = new address[](3);
        tradersKey = new uint256[](3);
        // 使用固定私钥便于测试签名
        tradersKey[0] = 0xA11CE;  // Alice
        tradersKey[1] = 0xB0B;    // Bob
        tradersKey[2] = 0xC0C;    // Carol

        for (uint256 i; i < traders.length; i++) {
            traders[i] = vm.addr(tradersKey[i]);
        }
    }

    /**
     * @notice 初始化 MetaNodeDealer 和永续合约市场
     * @dev 配置步骤：
     *      1. 设置最大持仓数量为 10
     *      2. 将测试合约设为订单发送者
     *      3. 创建 BTC 和 ETH 两个永续合约市场
     *      4. 配置风险参数
     *      5. 为交易者铸造代币并授权
     */
    function initMetaNodeDealer() public {
        metaNodeDealer.setMaxPositionAmount(10);
        metaNodeDealer.setOrderSender(address(this), true);
        
        // 创建两个永续合约市场
        for (uint256 i = 0; i < 2; i++) {
            Perpetual perp = new Perpetual(address(metaNodeDealer));
            TestMarkPriceSource priceSource = new TestMarkPriceSource();
            perpList.push(perp);
            priceSourceList.push(priceSource);
        }
        
        // ETH-PERP 风险参数
        // 10% 初始保证金 = 10 倍杠杆
        // 5% 维持保证金 = 清算阈值
        Types.RiskParams memory paramETH = Types.RiskParams({
            initialMarginRatio: 1e17,    // 10%
            liquidationThreshold: 5e16,   // 5%
            liquidationPriceOff: 1e16,    // 1% 清算折扣
            insuranceFeeRate: 2e16,       // 2% 保险费
            markPriceSource: address(priceSourceList[1]),
            name: "ETH",
            isRegistered: true
        });
        
        // BTC-PERP 风险参数
        // 5% 初始保证金 = 20 倍杠杆
        // 3% 维持保证金 = 清算阈值
        Types.RiskParams memory paramBTC = Types.RiskParams({
            initialMarginRatio: 5e16,     // 5%
            liquidationThreshold: 3e16,    // 3%
            liquidationPriceOff: 1e16,     // 1% 清算折扣
            insuranceFeeRate: 1e16,        // 1% 保险费
            markPriceSource: address(priceSourceList[0]),
            name: "BTC",
            isRegistered: true
        });
        
        // 注册永续合约市场
        metaNodeDealer.setPerpRiskParams(address(perpList[0]), paramBTC);
        metaNodeDealer.setPerpRiskParams(address(perpList[1]), paramETH);
        metaNodeDealer.setSecondaryAsset(address(musd));
        metaNodeDealer.setFundingRateKeeper(address(this));
        metaNodeDealer.setInsurance(insurance);
        
        // 为每个交易者铸造代币并授权
        for (uint256 i = 0; i < traders.length; i++) {
            usdc.mint(traders[i], 1_000_000e6);
            musd.mint(traders[i], 1_000_000e6);
            vm.startPrank(traders[i]);
            usdc.approve(address(metaNodeDealer), 1_000_000e6);
            musd.approve(address(metaNodeDealer), 1_000_000e6);
            vm.stopPrank();
        }
    }

    /**
     * @notice 构建签名订单
     * @param signer 签名者地址
     * @param privateKey 签名者私钥
     * @param paper 仓位数量（正=多头，负=空头）
     * @param credit 资金数量（与 paper 异号）
     * @param perpetual 永续合约地址
     * @return order 订单结构体
     * @return signature EIP-712 签名
     * 
     * @dev 签名流程：
     *      1. 构建订单结构体
     *      2. 计算 EIP-712 域分隔符
     *      3. 计算消息摘要
     *      4. 使用私钥签名
     */
    function buildOrder(
        address signer,
        uint256 privateKey,
        int128 paper,
        int128 credit,
        address perpetual
    )
        public
        view
        returns (Types.Order memory order, bytes memory signature)
    {
        // 设置手续费率
        int64 makerFeeRate = 1e14;   // 0.01%
        int64 takerFeeRate = 5e14;   // 0.05%
        
        // 打包订单信息字段
        bytes memory infoBytes = 
            abi.encodePacked(makerFeeRate, takerFeeRate, uint64(block.timestamp), uint64(block.timestamp));
        
        order = Types.Order({
            perp: perpetual,
            signer: signer,
            paperAmount: paper,
            creditAmount: credit,
            info: bytes32(infoBytes)
        });
        
        // 计算 EIP-712 签名
        bytes32 domainSeparator = EIP712Test._buildDomainSeparator("MetaNode", "1", address(metaNodeDealer));
        bytes32 digest = keccak256(abi.encodePacked("\x19\x01", domainSeparator, EIP712Test._structHash(order)));
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(privateKey, digest);
        signature = abi.encodePacked(r, s, v);
    }

    /**
     * @notice 构建交易数据
     * @param takerAmount Taker 的 paper 数量
     * @param takerCredit Taker 的 credit 数量
     * @param makerAmount Maker 的 paper 数量
     * @param makerCredit Maker 的 credit 数量
     * @param matchPaperAmount1 第一个订单的成交数量
     * @param matchPaperAmount2 第二个订单的成交数量
     * @param perpetual 永续合约地址
     * @return 编码后的交易数据
     */
    function constructTradeData(
        int128 takerAmount,
        int128 takerCredit,
        int128 makerAmount,
        int128 makerCredit,
        uint256 matchPaperAmount1,
        uint256 matchPaperAmount2,
        address perpetual
    )
        internal
        view
        returns (bytes memory)
    {
        // 构建 Taker 订单（traders[0]）
        (Types.Order memory order1, bytes memory signature1) =
            buildOrder(traders[0], tradersKey[0], takerAmount, takerCredit, perpetual);
        // 构建 Maker 订单（traders[1]）
        (Types.Order memory order2, bytes memory signature2) =
            buildOrder(traders[1], tradersKey[1], makerAmount, makerCredit, perpetual);
        
        Types.Order[] memory orderList = new Types.Order[](2);
        orderList[0] = order1;
        orderList[1] = order2;
        
        bytes[] memory signatureList = new bytes[](2);
        signatureList[0] = signature1;
        signatureList[1] = signature2;
        
        uint256[] memory matchPaperAmount = new uint256[](2);
        matchPaperAmount[0] = matchPaperAmount1;
        matchPaperAmount[1] = matchPaperAmount2;
        
        return abi.encode(orderList, signatureList, matchPaperAmount);
    }

    /**
     * @notice 执行交易
     * @dev 便捷函数，构建交易数据并执行
     */
    function trade(
        int128 takerAmount,
        int128 takerCredit,
        int128 makerAmount,
        int128 makerCredit,
        uint256 matchPaperAmount1,
        uint256 matchPaperAmount2,
        address perpetual
    )
        public
    {
        bytes memory tradeData = constructTradeData(
            takerAmount, takerCredit, makerAmount, makerCredit, matchPaperAmount1, matchPaperAmount2, perpetual
        );
        Perpetual(perpetual).trade(tradeData);
    }

    /**
     * @notice 测试环境初始化
     * @dev Foundry 会在每个测试前调用此函数
     *      初始化内容：
     *      1. 部署测试代币
     *      2. 创建测试用户
     *      3. 部署和配置 MetaNodeDealer
     *      4. 设置初始价格（BTC: $30,000, ETH: $2,000）
     */
    function setUp() public virtual {
        // 部署测试代币
        musd = new TestERC20("MUSD", "MUSD", 6);
        usdc = new TestERC20("USDC", "USDC", 6);
        
        // 初始化用户
        initUsers();
        
        // 部署 MetaNodeDealer
        metaNodeDealer = new MetaNodeDealer(address(usdc));
        initMetaNodeDealer();
        
        // 设置初始价格
        priceSourceList[0].setMarkPrice(30_000e6);  // BTC = $30,000
        priceSourceList[1].setMarkPrice(2000e6);    // ETH = $2,000
    }
}
