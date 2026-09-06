/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

import "../init/TradingInit.sol";
import "../utils/Checkers.sol";

/**
 * @title ScenarioTradingTest - 完整交易场景测试
 * @author MetaNode Team
 * @notice 演示永续合约交易的完整生命周期
 * 
 * ============================================================
 *                      测试场景概览
 * ============================================================
 * 
 * 场景1: 开仓和平仓（盈利）
 *   - Alice 做多 1 BTC @ $30,000
 *   - 价格涨到 $35,000
 *   - Alice 平仓获利 $5,000
 * 
 * 场景2: 开仓和平仓（亏损）
 *   - Bob 做多 1 BTC @ $30,000
 *   - 价格跌到 $28,000
 *   - Bob 平仓亏损 $2,000
 * 
 * 场景3: 多空对战
 *   - Alice 做多，Bob 做空
 *   - 价格变动后双方盈亏
 * 
 * 场景4: 部分平仓
 *   - Alice 开 2 BTC 多头
 *   - 先平 1 BTC，锁定部分利润
 *   - 再平剩余 1 BTC
 * 
 * ============================================================
 *                   前端开发参考
 * ============================================================
 * 
 * 订单构建流程：
 * 1. 获取用户私钥签名
 * 2. 构建 Types.Order 结构体
 * 3. 计算 EIP-712 签名
 * 4. 提交到后端撮合服务
 * 
 * 关键参数说明：
 * - paper: 正数=做多，负数=做空
 * - credit: 与 paper 符号相反（做多时为负）
 * - info: 包含手续费率和过期时间
 * 
 * ============================================================
 *                   后端开发参考
 * ============================================================
 * 
 * 撮合引擎职责：
 * 1. 收集用户签名订单
 * 2. 匹配买卖订单
 * 3. 构建 tradeData
 * 4. 调用 perpetual.trade(tradeData)
 * 
 * 结算数据格式：
 * tradeData = abi.encode(orderList, signatureList, matchPaperAmount)
 */
contract ScenarioTradingTest is Checkers {
    // ==================== 测试常量 ====================
    
    /// @notice 初始 BTC 价格：$30,000
    uint256 constant INITIAL_BTC_PRICE = 30_000e6;
    /// @notice 初始保证金：$10,000 USDC
    uint256 constant INITIAL_MARGIN = 10_000e6;
    
    /**
     * @notice 测试前准备
     * @dev 为交易者存入保证金
     */
    function setUp() public override {
        super.setUp();
        
        // traders[0] = Alice, traders[1] = Bob 存入保证金
        vm.startPrank(traders[0]);
        metaNodeDealer.deposit(INITIAL_MARGIN, 0, traders[0]);
        vm.stopPrank();
        
        vm.startPrank(traders[1]);
        metaNodeDealer.deposit(INITIAL_MARGIN, 0, traders[1]);
        vm.stopPrank();
    }
    
    // ==================== 场景1: 盈利平仓 ====================
    
    /**
     * @notice 场景1：做多盈利后平仓
     * @dev 
     * 交易流程：
     * 1. Alice 以 $30,000 做多 1 BTC（Taker）
     * 2. Bob 以 $30,000 做空 1 BTC（Maker）
     * 3. BTC 价格涨到 $35,000
     * 4. Alice 平仓（做空 1 BTC）获利
     * 5. Bob 平仓（做多 1 BTC）亏损
     * 
     * 预期结果：
     * - Alice 盈利约 $5,000（减去手续费）
     * - Bob 亏损约 $5,000（加上手续费）
     * 
     * 手续费计算：
     * - Taker Fee: 0.05% × $30,000 = $15
     * - Maker Fee: 0.01% × $30,000 = $3
     */
    function testScenario1_LongProfitClose() public {
        // ========== 步骤1: 开仓 ==========
        // Alice 做多 1 BTC: paper=+1, credit=-30000
        // Bob 做空 1 BTC: paper=-1, credit=+30000
        trade(
            1e18,           // Alice paper: +1 BTC
            -30_000e6,      // Alice credit: -$30,000
            -1e18,          // Bob paper: -1 BTC
            30_000e6,       // Bob credit: +$30,000
            1e18,           // 成交数量
            1e18,           // 成交数量
            address(perpList[0])
        );
        
        // 验证仓位
        (int256 alicePaper, int256 aliceCredit) = perpList[0].balanceOf(traders[0]);
        (int256 bobPaper, int256 bobCredit) = perpList[0].balanceOf(traders[1]);
        
        assertEq(alicePaper, 1e18, "Alice should have 1 BTC long");
        assertEq(bobPaper, -1e18, "Bob should have 1 BTC short");
        
        // ========== 步骤2: 价格上涨 ==========
        priceSourceList[0].setMarkPrice(35_000e6);  // BTC 涨到 $35,000
        
        // 验证风险状态
        (int256 aliceNetValue,,,) = metaNodeDealer.getTraderRisk(traders[0]);
        (int256 bobNetValue,,,) = metaNodeDealer.getTraderRisk(traders[1]);
        
        // Alice 应该盈利，Bob 应该亏损
        assertTrue(aliceNetValue > int256(INITIAL_MARGIN), "Alice should be in profit");
        assertTrue(bobNetValue < int256(INITIAL_MARGIN), "Bob should be in loss");
        
        // ========== 步骤3: 平仓 ==========
        // Alice 平仓做空 1 BTC @ $35,000
        // Bob 平仓做多 1 BTC @ $35,000
        trade(
            -1e18,          // Alice paper: -1 BTC（平仓）
            35_000e6,       // Alice credit: +$35,000
            1e18,           // Bob paper: +1 BTC（平仓）
            -35_000e6,      // Bob credit: -$35,000
            1e18,
            1e18,
            address(perpList[0])
        );
        
        // 验证仓位已清空
        (alicePaper, aliceCredit) = perpList[0].balanceOf(traders[0]);
        (bobPaper, bobCredit) = perpList[0].balanceOf(traders[1]);
        
        assertEq(alicePaper, 0, "Alice position should be closed");
        assertEq(bobPaper, 0, "Bob position should be closed");
        
        // ========== 步骤4: 验证盈亏 ==========
        (int256 aliceFinalCredit,,,,) = metaNodeDealer.getCreditOf(traders[0]);
        (int256 bobFinalCredit,,,,) = metaNodeDealer.getCreditOf(traders[1]);
        
        int256 alicePnL = aliceFinalCredit - int256(INITIAL_MARGIN);
        int256 bobPnL = bobFinalCredit - int256(INITIAL_MARGIN);
        
        // Alice 应该盈利约 $5000（减去手续费）
        assertTrue(alicePnL > 4900e6, "Alice should profit ~$5000");
        // Bob 应该亏损约 $5000（加上手续费）
        assertTrue(bobPnL < -4900e6, "Bob should lose ~$5000");
    }
    
    // ==================== 场景2: 亏损平仓 ====================
    
    /**
     * @notice 场景2：做多亏损后平仓
     * @dev 
     * 交易流程：
     * 1. Alice 以 $30,000 做多 1 BTC
     * 2. BTC 价格跌到 $28,000
     * 3. Alice 平仓亏损
     */
    function testScenario2_LongLossClose() public {
        // 开仓
        trade(
            1e18, -30_000e6,   // Alice 做多
            -1e18, 30_000e6,   // Bob 做空
            1e18, 1e18,
            address(perpList[0])
        );
        
        // 价格下跌
        priceSourceList[0].setMarkPrice(28_000e6);  // $28,000
        
        // 平仓
        trade(
            -1e18, 28_000e6,   // Alice 平仓
            1e18, -28_000e6,   // Bob 平仓
            1e18, 1e18,
            address(perpList[0])
        );
        
        // 验证结果
        (int256 aliceFinal,,,,) = metaNodeDealer.getCreditOf(traders[0]);
        (int256 bobFinal,,,,) = metaNodeDealer.getCreditOf(traders[1]);
        
        int256 alicePnL = aliceFinal - int256(INITIAL_MARGIN);
        int256 bobPnL = bobFinal - int256(INITIAL_MARGIN);
        
        // Alice 亏损约 $2000
        assertTrue(alicePnL < -1900e6, "Alice should lose ~$2000");
        // Bob 盈利约 $2000
        assertTrue(bobPnL > 1900e6, "Bob should profit ~$2000");
    }
    
    // ==================== 场景3: 多空对战 ====================
    
    /**
     * @notice 场景3：多空双方博弈
     * @dev 演示零和游戏特性
     * 
     * 永续合约是零和游戏：
     * Alice 的盈利 = Bob 的亏损（忽略手续费）
     */
    function testScenario3_LongVsShort() public {
        // 初始状态
        (int256 aliceInit,,,,) = metaNodeDealer.getCreditOf(traders[0]);
        (int256 bobInit,,,,) = metaNodeDealer.getCreditOf(traders[1]);
        
        // 开仓：Alice 做多，Bob 做空
        trade(
            2e18, -60_000e6,   // Alice 做多 2 BTC
            -2e18, 60_000e6,   // Bob 做空 2 BTC
            2e18, 2e18,
            address(perpList[0])
        );
        
        // 价格波动
        priceSourceList[0].setMarkPrice(32_000e6);  // $32,000
        
        // 平仓
        trade(
            -2e18, 64_000e6,   // Alice 平仓
            2e18, -64_000e6,   // Bob 平仓
            2e18, 2e18,
            address(perpList[0])
        );
        
        // 最终盈亏
        (int256 aliceFinal,,,,) = metaNodeDealer.getCreditOf(traders[0]);
        (int256 bobFinal,,,,) = metaNodeDealer.getCreditOf(traders[1]);
        
        int256 alicePnL = aliceFinal - aliceInit;
        int256 bobPnL = bobFinal - bobInit;
        int256 totalPnL = alicePnL + bobPnL;
        
        // 验证零和特性（总和应该约等于负手续费）
        assertTrue(totalPnL < 0, "Total P&L should be negative (fees)");
        assertTrue(totalPnL > -200e6, "Fees should be reasonable");
    }
    
    // ==================== 场景4: 部分平仓 ====================
    
    /**
     * @notice 场景4：部分平仓锁定利润
     * @dev 
     * 交易流程：
     * 1. Alice 做多 2 BTC
     * 2. 价格上涨，平仓 1 BTC 锁定部分利润
     * 3. 价格继续上涨，平仓剩余 1 BTC
     * 
     * 这是常见的交易策略：分批止盈
     */
    function testScenario4_PartialClose() public {
        // 开仓 2 BTC
        trade(
            2e18, -60_000e6,
            -2e18, 60_000e6,
            2e18, 2e18,
            address(perpList[0])
        );
        
        // 价格涨到 $33,000
        priceSourceList[0].setMarkPrice(33_000e6);
        
        // 平仓 1 BTC
        trade(
            -1e18, 33_000e6,
            1e18, -33_000e6,
            1e18, 1e18,
            address(perpList[0])
        );
        
        // 验证剩余仓位
        (int256 alicePaper,) = perpList[0].balanceOf(traders[0]);
        assertEq(alicePaper, 1e18, "Should have 1 BTC remaining");
        
        // 价格继续涨到 $36,000
        priceSourceList[0].setMarkPrice(36_000e6);
        
        // 平仓剩余 1 BTC
        trade(
            -1e18, 36_000e6,
            1e18, -36_000e6,
            1e18, 1e18,
            address(perpList[0])
        );
        
        // 最终盈亏
        (int256 aliceFinal,,,,) = metaNodeDealer.getCreditOf(traders[0]);
        int256 alicePnL = aliceFinal - int256(INITIAL_MARGIN);
        
        // 预期盈利：(33000-30000) + (36000-30000) = 3000 + 6000 = $9000
        assertTrue(alicePnL > 8800e6, "Should profit ~$9000");
    }
}
