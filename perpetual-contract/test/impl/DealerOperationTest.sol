/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

import "../init/TradingInit.sol";
import "../utils/Checkers.sol";

/**
 * @title OperationTest - 管理操作测试
 * @notice 测试 MetaNodeDealer 的管理员操作函数
 * 
 * 测试覆盖：
 * 1. 市场注销功能
 * 2. 权限控制检查
 * 3. 风险参数验证
 * 4. 次级资产设置限制
 */
contract OperationTest is Checkers {
    /**
     * @notice 测试注销永续合约市场
     * @dev 验证：
     *      1. 设置 isRegistered = false 可以注销市场
     *      2. 注销后已注册市场列表长度减少
     */
    function testRemovePerp() public {
        // 构建注销参数
        Types.RiskParams memory paramETH2 = Types.RiskParams({
            initialMarginRatio: 1e17,
            liquidationThreshold: 5e16,
            liquidationPriceOff: 1e16,
            insuranceFeeRate: 1e16,
            markPriceSource: address(priceSourceList[1]),
            name: "ETH",
            isRegistered: false  // 设为 false 以注销
        });
        metaNodeDealer.setPerpRiskParams(address(perpList[1]), paramETH2);
        
        // 验证市场已被移除
        address[] memory perps2 = metaNodeDealer.getAllRegisteredPerps();
        assertEq(perps2.length, 1);  // 原来 2 个，现在只剩 1 个
    }

    /**
     * @notice 测试仅已注册永续合约可调用的函数
     * @dev 验证未注册的合约调用以下函数会 revert：
     *      - approveTrade
     *      - requestLiquidation
     *      - openPosition
     *      - realizePnl
     */
    function testOnlyRegisteredPerp() public {
        cheats.expectRevert("META_PERP_NOT_REGISTERED");
        metaNodeDealer.approveTrade(traders[0], "0x00");

        cheats.expectRevert("META_PERP_NOT_REGISTERED");
        metaNodeDealer.requestLiquidation(traders[0], traders[1], traders[0], 0);

        cheats.expectRevert("META_PERP_NOT_REGISTERED");
        metaNodeDealer.openPosition(traders[0]);

        cheats.expectRevert("META_PERP_NOT_REGISTERED");
        metaNodeDealer.realizePnl(traders[0], 0);
    }

    /**
     * @notice 测试无效风险参数
     * @dev 验证：liquidationPriceOff + insuranceFeeRate > liquidationThreshold 时会 revert
     *      这是为了确保清算折扣和保险费之和不超过清算阈值
     */
    function testInvalidRiskParam() public {
        Types.RiskParams memory paramBTC2 = Types.RiskParams({
            initialMarginRatio: 5e16,
            liquidationThreshold: 3e16,      // 3%
            liquidationPriceOff: 2e16,       // 2%
            insuranceFeeRate: 2e16,          // 2%  -> 2% + 2% > 3% 无效
            markPriceSource: address(priceSourceList[0]),
            name: "BTC",
            isRegistered: true
        });
        cheats.expectRevert("META_INVALID_RISK_PARAM");
        metaNodeDealer.setPerpRiskParams(address(perpList[0]), paramBTC2);
    }

    /**
     * @notice 测试次级资产不能重复设置
     * @dev 验证已设置次级资产后再次设置会 revert
     */
    function testSecondaryAssetCanNotChange() public {
        cheats.expectRevert("META_SECONDARY_ASSET_ALREADY_EXIST");
        metaNodeDealer.setSecondaryAsset(address(perpList[0]));
        // 同时测试禁用快速取款
        metaNodeDealer.disableFastWithdraw(true);
    }

    /**
     * @notice 测试设置次级资产精度检查
     * @dev 验证：次级资产小数位数必须与主资产一致
     */
    function testSetSecondary() public {
        MetaNodeDealer jd = new MetaNodeDealer(address(usdc));
        // 创建一个 20 位小数的代币（USDC 是 6 位）
        TestERC20 fake = new TestERC20("fake", "fake", 20);
        cheats.expectRevert("META_SECONDARY_ASSET_DECIMAL_WRONG");
        jd.setSecondaryAsset(address(fake));
    }
}
