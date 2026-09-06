/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

import "../init/TradingInit.sol";

/**
 * @title Checkers - 测试断言工具
 * @notice 提供用于测试的断言和验证函数
 * 
 * 功能：
 * 1. 验证交易者的资金状态
 * 2. 比较预期值和实际值
 */
contract Checkers is TradingInit {
    // 排除在覆盖率报告之外
    function testC() public { }

    /**
     * @notice 资金详情结构体
     * @dev 用于存储和返回交易者的完整资金状态
     */
    struct Credit {
        /// @notice 主资产余额（可为负）
        int256 primaryCredit;
        /// @notice 次级资产余额
        uint256 secondaryCredit;
        /// @notice 待取款主资产数量
        uint256 pendingPrimaryWithdraw;
        /// @notice 待取款次级资产数量
        uint256 pendingSecondaryWithdraw;
        /// @notice 取款可执行时间戳
        uint256 executionTimestamp;
    }

    /**
     * @notice 余额结构体
     * @dev 用于存储仓位和资金信息
     */
    struct Balance {
        /// @notice 仓位数量
        uint256 paper;
        /// @notice 资金数量
        uint256 credit;
    }

    /**
     * @notice 检查并验证交易者的资金状态
     * @param trader 交易者地址
     * @param primary 期望的主资产余额
     * @param secondary 期望的次级资产余额
     * @return credit 实际的资金详情
     * @dev 会自动断言主资产和次级资产余额是否符合预期
     */
    function checkCredit(address trader, int256 primary, uint256 secondary) public returns (Credit memory credit) {
        (
            int256 primaryCredit,
            uint256 secondaryCredit,
            uint256 pendingPrimaryWithdraw,
            uint256 pendingSecondaryWithdraw,
            uint256 executionTimestamp
        ) = metaNodeDealer.getCreditOf(trader);

        credit.primaryCredit = primaryCredit;
        credit.secondaryCredit = secondaryCredit;
        credit.pendingPrimaryWithdraw = pendingPrimaryWithdraw;
        credit.pendingSecondaryWithdraw = pendingSecondaryWithdraw;
        credit.executionTimestamp = executionTimestamp;

        // 断言余额符合预期
        assertEq(credit.primaryCredit, primary);
        assertEq(credit.secondaryCredit, secondary);
    }
}
