/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

import "../libraries/Errors.sol";
import "./Types.sol";

/**
 * @title Position - 仓位管理库
 * @notice 处理用户仓位的开仓记录和平仓结算
 * 
 * 核心功能：
 * 1. 开仓记录：当用户在新市场开仓时，记录到持仓列表
 * 2. 平仓结算：当用户完全平仓时，实现盈亏并从列表移除
 * 
 * 设计说明：
 * - openPositions 记录用户持有仓位的市场列表
 * - positionSerialNum 用于链下盈亏计算，每次平仓+1
 * - msg.sender 是调用此函数的 Perpetual 合约
 */
library Position {
    // ==================== 仓位注册 ====================

    /**
     * @notice 开仓注册
     * @param state 系统状态
     * @param trader 交易者地址
     * @dev 当交易者在某市场首次开仓时，将该市场添加到持仓列表
     *      msg.sender 是调用此函数的 Perpetual 合约
     * 
     * 限制说明：
     * - maxPositionAmount 限制用户同时持仓的市场数量
     * - 防止 gas 消耗过大（遍历持仓列表计算风险）
     */
    function _openPosition(Types.State storage state, address trader) internal {
        require(state.openPositions[trader].length < state.maxPositionAmount, Errors.POSITION_AMOUNT_REACH_UPPER_LIMIT);
        state.openPositions[trader].push(msg.sender);
    }

    /**
     * @notice 实现盈亏并移除仓位
     * @param state 系统状态
     * @param trader 交易者地址
     * @param pnl 盈亏金额（正=盈利，负=亏损）
     * @dev 当交易者完全平仓时调用：
     *      1. 将盈亏计入 primaryCredit
     *      2. 增加仓位序列号（用于链下追踪）
     *      3. 从持仓列表中移除该市场
     * 
     * 实现细节：
     * - 使用 swap-and-pop 方式移除，节省 gas
     * - pnl 实际上是 Perpetual 合约中的 reducedCredit
     */
    function _realizePnl(Types.State storage state, address trader, int256 pnl) internal {
        // 将盈亏计入账户余额
        state.primaryCredit[trader] += pnl;
        // 增加序列号，用于链下区分不同轮次的仓位
        state.positionSerialNum[trader][msg.sender] += 1;

        // 从持仓列表中移除该市场（swap-and-pop）
        address[] storage positionList = state.openPositions[trader];
        for (uint256 i = 0; i < positionList.length;) {
            if (positionList[i] == msg.sender) {
                // 将最后一个元素移到当前位置，然后删除最后一个
                positionList[i] = positionList[positionList.length - 1];
                positionList.pop();
                break;
            }
            unchecked {
                ++i;
            }
        }
    }
}
