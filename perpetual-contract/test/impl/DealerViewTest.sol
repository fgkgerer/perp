/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

import "../init/TradingInit.sol";
import "../utils/Checkers.sol";

/**
 * @title MetaNodeViewTest - 视图函数测试
 * @notice 测试 MetaNodeDealer 的视图查询函数
 * 
 * 测试覆盖：
 * 1. 订单发送者验证
 * 2. 快速取款验证
 * 3. 授权额度查询
 * 4. 订单成交量查询
 * 5. 编码辅助函数
 * 6. 安全状态检查
 * 7. 版本号查询
 */
contract MetaNodeViewTest is Checkers {
    /**
     * @notice 测试视图函数基本功能
     * @dev 验证各个查询函数能正常工作
     */
    function testMetaNodeView() public {
        // 测试订单发送者验证（setUp 中已设置 address(this) 为有效）
        bool ifOrderValid = metaNodeDealer.isOrderSenderValid(address(this));
        assertEq(ifOrderValid, true);
        
        // 测试快速取款验证（默认不在白名单）
        bool ifWithdrawValid = metaNodeDealer.isFastWithdrawalValid(address(this));
        assertEq(ifWithdrawValid, false);

        // 测试授权额度查询（默认为 0）
        (uint256 primaryCreditAllowed,) = metaNodeDealer.isCreditAllowed(traders[0], address(this));
        assertEq(primaryCreditAllowed, 0);

        // 测试订单成交量查询
        metaNodeDealer.getOrderFilledAmount(Types.ORDER_TYPEHASH);
        
        // 测试编码辅助函数
        metaNodeDealer.getRequestWithdrawCallData(address(this), 0, 0);
        metaNodeDealer.getExecuteWithdrawCallData(address(this), address(this), false, "");
        
        // 测试初始保证金安全检查
        metaNodeDealer.isIMSafe(address(this));
    }

    /**
     * @notice 测试版本号查询
     * @dev 验证 version() 函数可以正常调用
     */
    function testVersion() public view {
        metaNodeDealer.version();
    }
}
