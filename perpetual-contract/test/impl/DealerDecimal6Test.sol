/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

import "../init/TradingInit.sol";

/**
 * @title Decimal6Test - 6位小数精度测试
 * @notice 测试 MetaNodeDealer 在使用 6 位小数（如 USDC）时的正确性
 * 
 * 测试场景：
 * 1. 存款后余额检查
 * 2. 交易后净值和敞口计算
 * 3. 清算价格计算
 * 4. 次级资产重复设置限制
 */
contract Decimal6Test is TradingInit {
    /**
     * @notice 存款辅助函数
     * @dev 两个交易者各存入 10,000 MUSD（次级资产）
     */
    function deposit() public {
        vm.startPrank(traders[0]);
        metaNodeDealer.deposit(0, 10_000e6, traders[0]);
        vm.stopPrank();
        vm.startPrank(traders[1]);
        metaNodeDealer.deposit(0, 10_000e6, traders[1]);
        vm.stopPrank();
    }

    /**
     * @notice 测试余额检查
     * @dev 验证：
     *      1. 交易后次级资产余额不变
     *      2. 净值正确计算（包含手续费影响）
     *      3. 敞口正确计算
     * 
     * 交易详情：
     * - traders[0] 以 $30,000 做多 1 BTC
     * - traders[1] 以 $30,000 做空 1 BTC
     */
    function testBalanceCheck() public {
        deposit();
        // 执行交易：做多 1 BTC @ $30,000
        trade(1e18, -30_000e6, -1e18, 30_000e6, 1e18, 1e18, address(perpList[0]));
        
        // 验证次级资产余额不变
        (, uint256 secondaryCredit0,,,) = metaNodeDealer.getCreditOf(traders[0]);
        (, uint256 secondaryCredit1,,,) = metaNodeDealer.getCreditOf(traders[1]);
        assertEq(secondaryCredit0, 10_000e6);
        assertEq(secondaryCredit1, 10_000e6);
        
        // 验证净值和敞口
        (int256 netValue0, uint256 exposure0,,) = metaNodeDealer.getTraderRisk(traders[0]);
        (int256 netValue1, uint256 exposure1,,) = metaNodeDealer.getTraderRisk(traders[1]);
        // 净值 = 10000 - 手续费
        assertEq(netValue0, 9985e6);  // Taker 手续费更高
        assertEq(netValue1, 9997e6);  // Maker 手续费较低
        // 敞口 = 1 BTC * $30,000
        assertEq(exposure0, 30_000e6);
        assertEq(exposure1, 30_000e6);
    }

    /**
     * @notice 测试清算价格计算
     * @dev 验证：
     *      1. 无仓位时清算价格为 0
     *      2. 有仓位后清算价格正确计算
     *      3. 价格触及清算价时账户不安全
     * 
     * 清算价格公式：
     * 多头：当价格下跌使净值 < 维持保证金时触发
     * 空头：当价格上涨使净值 < 维持保证金时触发
     */
    function testLiqPrice() public {
        deposit();
        // 无仓位时清算价格应为 0
        metaNodeDealer.getLiquidationPrice(traders[0], address(perpList[0]));
        
        // 执行交易
        trade(1e18, -30_000e6, -1e18, 30_000e6, 1e18, 1e18, address(perpList[0]));
        
        // 获取清算价格
        uint256 liquidationPrice0 = metaNodeDealer.getLiquidationPrice(traders[0], address(perpList[0]));
        uint256 liquidationPrice1 = metaNodeDealer.getLiquidationPrice(traders[1], address(perpList[0]));
        metaNodeDealer.getLiquidationPrice(traders[0], address(perpList[1]));
        
        // 验证清算价格
        assertEq(liquidationPrice0, 20_634_020_618);  // 多头清算价 ≈ $20,634
        assertEq(liquidationPrice1, 38_832_038_834);  // 空头清算价 ≈ $38,832
        
        // 测试价格触及清算价时账户变为不安全
        priceSourceList[0].setMarkPrice(20_000e6);  // BTC 跌到 $20,000
        assertEq(metaNodeDealer.isSafe(traders[0]), false);  // 多头被清算
        
        priceSourceList[0].setMarkPrice(40_000e6);  // BTC 涨到 $40,000
        assertEq(metaNodeDealer.isSafe(traders[1]), false);  // 空头被清算
    }

    /**
     * @notice 测试次级资产不能重复设置
     * @dev 验证次级资产一旦设置后不能更改
     */
    function testSecondaryAssetExist() public {
        deposit();
        trade(1e18, -30_000e6, -1e18, 30_000e6, 1e18, 1e18, address(perpList[0]));
        TestERC20 usdw = new TestERC20("USDW", "USDW", 12);
        cheats.expectRevert("META_SECONDARY_ASSET_ALREADY_EXIST");
        metaNodeDealer.setSecondaryAsset(address(usdw));
    }
}
