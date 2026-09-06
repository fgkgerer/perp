/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import "@openzeppelin/contracts/utils/math/SafeCast.sol";
import "../interfaces/internal/IPriceSource.sol";
import "../interfaces/IPerpetual.sol";
import "../libraries/Errors.sol";
import "../libraries/SignedDecimalMath.sol";
import "./Liquidation.sol";
import "./Operation.sol";
import "./Types.sol";

/**
 * @title Funding - 资金管理库
 * @notice 处理用户资金的存款、取款和转账操作
 * 
 * 资金流程说明：
 * 1. 存款（deposit）：用户将资产转入合约，获得交易余额
 * 2. 取款有两种模式：
 *    - 普通取款：requestWithdraw -> 等待时间锁 -> executeWithdraw
 *    - 快速取款：直接调用 fastWithdraw（需要权限）
 * 
 * 安全机制：
 * - 取款时间锁：防止在订单成交前取走保证金
 * - 授权额度：允许操作员在限额内操作用户资金
 * - 安全检查：取款后必须满足保证金要求
 */
library Funding {
    using SafeERC20 for IERC20;

    // ==================== 事件 ====================

    /// @notice 存款事件
    /// @param to 存入的目标账户
    /// @param payer 支付资产的地址
    /// @param primaryAmount 主资产数量
    /// @param secondaryAmount 次级资产数量
    event Deposit(address indexed to, address indexed payer, uint256 primaryAmount, uint256 secondaryAmount);

    /// @notice 取款事件
    /// @param to 资金接收地址
    /// @param payer 取款来源账户
    /// @param primaryAmount 主资产数量
    /// @param secondaryAmount 次级资产数量
    event Withdraw(address indexed to, address indexed payer, uint256 primaryAmount, uint256 secondaryAmount);

    /// @notice 请求取款事件
    /// @param payer 取款来源账户
    /// @param primaryAmount 主资产数量
    /// @param secondaryAmount 次级资产数量
    /// @param executionTimestamp 可执行的时间戳
    event RequestWithdraw(
        address indexed payer, uint256 primaryAmount, uint256 secondaryAmount, uint256 executionTimestamp
    );

    /// @notice 内部转入事件（账户间转账）
    event TransferIn(address trader, uint256 primaryAmount, uint256 secondaryAmount);

    /// @notice 内部转出事件
    event TransferOut(address trader, uint256 primaryAmount, uint256 secondaryAmount);

    // ==================== 存款 ====================

    /**
     * @notice 存入保证金
     * @param state 系统状态
     * @param primaryAmount 主资产数量
     * @param secondaryAmount 次级资产数量
     * @param to 存入的目标账户
     * @dev 调用者需要提前 approve 足够的额度
     */
    function deposit(Types.State storage state, uint256 primaryAmount, uint256 secondaryAmount, address to) external {
        // 转入主资产并增加余额
        if (primaryAmount > 0) {
            IERC20(state.primaryAsset).safeTransferFrom(msg.sender, address(this), primaryAmount);
            state.primaryCredit[to] += SafeCast.toInt256(primaryAmount);
        }
        // 转入次级资产并增加余额
        if (secondaryAmount > 0) {
            IERC20(state.secondaryAsset).safeTransferFrom(msg.sender, address(this), secondaryAmount);
            state.secondaryCredit[to] += secondaryAmount;
        }
        emit Deposit(to, msg.sender, primaryAmount, secondaryAmount);
    }

    // ==================== 取款 ====================

    /**
     * @notice 检查取款请求是否有效
     * @param state 系统状态
     * @param spender 操作者地址
     * @param from 取款来源账户
     * @param primaryAmount 主资产数量
     * @param secondaryAmount 次级资产数量
     * @return 是否有效
     * @dev 如果操作者不是账户本人，检查授权额度
     */
    function isWithdrawValid(
        Types.State storage state,
        address spender,
        address from,
        uint256 primaryAmount,
        uint256 secondaryAmount
    )
        internal
        view
        returns (bool)
    {
        // 本人操作直接返回 true
        if (spender != from) {
            // 检查授权额度是否足够
            return (
                state.primaryCreditAllowed[from][spender] >= primaryAmount
                    && state.secondaryCreditAllowed[from][spender] >= secondaryAmount
            ) && (state.primaryCreditAllowed[from][spender] != 0 || state.secondaryCreditAllowed[from][spender] != 0);
        }
        return true;
    }

    /**
     * @notice 请求取款（第一步）
     * @param state 系统状态
     * @param from 取款来源账户
     * @param primaryAmount 主资产数量
     * @param secondaryAmount 次级资产数量
     * @dev 设置待取款金额和可执行时间
     *      时间锁机制确保链下订单有足够时间结算
     */
    function requestWithdraw(
        Types.State storage state,
        address from,
        uint256 primaryAmount,
        uint256 secondaryAmount
    )
        external
    {
        require(isWithdrawValid(state, msg.sender, from, primaryAmount, secondaryAmount), Errors.WITHDRAW_INVALID);
        // 记录待取款金额
        state.pendingPrimaryWithdraw[from] = primaryAmount;
        state.pendingSecondaryWithdraw[from] = secondaryAmount;
        // 设置可执行时间 = 当前时间 + 时间锁
        state.withdrawExecutionTimestamp[from] = block.timestamp + state.withdrawTimeLock;
        emit RequestWithdraw(from, primaryAmount, secondaryAmount, state.withdrawExecutionTimestamp[from]);
    }

    /**
     * @notice 执行取款（第二步）
     * @param state 系统状态
     * @param from 取款来源账户
     * @param to 资金接收地址
     * @param isInternal 是否为内部转账（不实际转出合约）
     * @param param 回调参数（可选）
     * @dev 在时间锁到期后执行实际的资金转出
     */
    function executeWithdraw(
        Types.State storage state,
        address from,
        address to,
        bool isInternal,
        bytes memory param
    )
        external
    {
        // 检查时间锁是否到期
        require(state.withdrawExecutionTimestamp[from] <= block.timestamp, Errors.WITHDRAW_PENDING);
        uint256 primaryAmount = state.pendingPrimaryWithdraw[from];
        uint256 secondaryAmount = state.pendingSecondaryWithdraw[from];
        require(isWithdrawValid(state, msg.sender, from, primaryAmount, secondaryAmount), Errors.WITHDRAW_INVALID);
        // 清空待取款金额（无需重置时间戳，因为金额已清零）
        state.pendingPrimaryWithdraw[from] = 0;
        state.pendingSecondaryWithdraw[from] = 0;
        _withdraw(state, msg.sender, from, to, primaryAmount, secondaryAmount, isInternal, param);
    }

    /**
     * @notice 快速取款（一步完成）
     * @param state 系统状态
     * @param from 取款来源账户
     * @param to 资金接收地址
     * @param primaryAmount 主资产数量
     * @param secondaryAmount 次级资产数量
     * @param isInternal 是否为内部转账
     * @param param 回调参数
     * @dev 跳过时间锁直接取款，仅限：
     *      1. 快速取款未禁用时，或
     *      2. 调用者在快速取款白名单中
     */
    function fastWithdraw(
        Types.State storage state,
        address from,
        address to,
        uint256 primaryAmount,
        uint256 secondaryAmount,
        bool isInternal,
        bytes memory param
    )
        external
    {
        // 检查快速取款权限
        require(
            !state.fastWithdrawDisabled || state.fastWithdrawalWhitelist[msg.sender], Errors.FAST_WITHDRAW_NOT_ALLOWED
        );
        require(isWithdrawValid(state, msg.sender, from, primaryAmount, secondaryAmount), Errors.WITHDRAW_INVALID);
        _withdraw(state, msg.sender, from, to, primaryAmount, secondaryAmount, isInternal, param);
    }

    /**
     * @notice 内部取款实现
     * @param state 系统状态
     * @param spender 操作者
     * @param from 取款来源
     * @param to 接收地址
     * @param primaryAmount 主资产数量
     * @param secondaryAmount 次级资产数量
     * @param isInternal 是否内部转账
     * @param param 回调参数
     * @dev 核心逻辑：
     *      1. 扣减授权额度（如果是代理操作）
     *      2. 扣减余额
     *      3. 转移资产（内部转账或外部转出）
     *      4. 安全检查
     *      5. 可选的回调执行
     */
    function _withdraw(
        Types.State storage state,
        address spender,
        address from,
        address to,
        uint256 primaryAmount,
        uint256 secondaryAmount,
        bool isInternal,
        bytes memory param
    )
        private
    {
        // 如果是代理操作，扣减授权额度
        if (spender != from) {
            state.primaryCreditAllowed[from][spender] -= primaryAmount;
            state.secondaryCreditAllowed[from][spender] -= secondaryAmount;
            emit Operation.FundOperatorAllowedChange(
                from, spender, state.primaryCreditAllowed[from][spender], state.secondaryCreditAllowed[from][spender]
            );
        }

        // 处理主资产
        if (primaryAmount > 0) {
            state.primaryCredit[from] -= SafeCast.toInt256(primaryAmount);
            if (isInternal) {
                // 内部转账：只改余额，不转移实际资产
                state.primaryCredit[to] += SafeCast.toInt256(primaryAmount);
            } else {
                // 外部取款：实际转出资产
                IERC20(state.primaryAsset).safeTransfer(to, primaryAmount);
            }
        }

        // 处理次级资产
        if (secondaryAmount > 0) {
            state.secondaryCredit[from] -= secondaryAmount;
            if (isInternal) {
                state.secondaryCredit[to] += secondaryAmount;
            } else {
                IERC20(state.secondaryAsset).safeTransfer(to, secondaryAmount);
            }
        }

        // 安全检查：确保取款后账户仍满足保证金要求
        if (primaryAmount > 0) {
            // 取出主资产时使用更严格的检查（Solid IM Safe）
            require(Liquidation._isSolidIMSafe(state, from), Errors.ACCOUNT_NOT_SAFE);
        } else {
            // 只取次级资产时使用普通检查
            require(Liquidation._isIMSafe(state, from), Errors.ACCOUNT_NOT_SAFE);
        }

        // 触发相应事件
        if (isInternal) {
            emit TransferIn(to, primaryAmount, secondaryAmount);
            emit TransferOut(from, primaryAmount, secondaryAmount);
        } else {
            emit Withdraw(to, from, primaryAmount, secondaryAmount);
        }

        // 如果有回调参数，执行回调
        if (param.length != 0) {
            require(Address.isContract(to), "target is not a contract");
            require(state.isWithdrawalWhitelist[to], "target is not in whiteList");
            (bool success,) = to.call(param);
            if (success == false) {
                // 回调失败，转发错误信息
                assembly {
                    let ptr := mload(0x40)
                    let size := returndatasize()
                    returndatacopy(ptr, 0, size)
                    revert(ptr, size)
                }
            }
        }
    }
}
