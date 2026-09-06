/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import "@openzeppelin/contracts/utils/math/SafeCast.sol";
import "./interfaces/IDealer.sol";
import "./libraries/Errors.sol";
import "./libraries/Liquidation.sol";
import "./libraries/Trading.sol";
import "./MetaNodeStorage.sol";

/**
 * @title MetaNodeView - 永续合约系统视图函数
 * @notice 包含所有只读查询函数，用于获取系统状态和用户信息
 * 
 * 主要查询类型：
 * 1. 市场信息：风险参数、标记价格、资金费率
 * 2. 账户信息：余额、持仓、订单状态
 * 3. 风险信息：保证金率、清算价格、安全状态
 * 4. 权限信息：操作员、白名单状态
 */
abstract contract MetaNodeView is MetaNodeStorage, IDealer {
    // ==================== 基础状态查询 ====================

    /**
     * @notice 获取永续合约的风险参数
     * @param perp 永续合约地址
     * @return params 风险参数结构体
     */
    function getRiskParams(address perp) external view returns (Types.RiskParams memory params) {
        params = state.perpRiskParams[perp];
    }

    /**
     * @notice 获取所有已注册的永续合约地址
     * @return 永续合约地址数组
     */
    function getAllRegisteredPerps() external view returns (address[] memory) {
        return state.registeredPerp;
    }

    /**
     * @notice 获取永续合约的标记价格
     * @param perp 永续合约地址
     * @return 标记价格（18位小数）
     * @dev 标记价格用于计算未实现盈亏和保证金率
     *      通常是指数价格加上基差
     */
    function getMarkPrice(address perp) external view returns (uint256) {
        return Liquidation.getMarkPrice(state, perp);
    }

    /**
     * @notice 获取交易者的持仓市场列表
     * @param trader 交易者地址
     * @return 持仓的永续合约地址数组
     */
    function getPositions(address trader) external view returns (address[] memory) {
        return state.openPositions[trader];
    }

    /**
     * @notice 获取交易者的资金详情
     * @param trader 交易者地址
     * @return primaryCredit 主资产余额（可为负，表示亏损）
     * @return secondaryCredit 次级资产余额
     * @return pendingPrimaryWithdraw 待取款的主资产数量
     * @return pendingSecondaryWithdraw 待取款的次级资产数量
     * @return executionTimestamp 取款可执行的时间戳
     */
    function getCreditOf(address trader)
        external
        view
        returns (
            int256 primaryCredit,
            uint256 secondaryCredit,
            uint256 pendingPrimaryWithdraw,
            uint256 pendingSecondaryWithdraw,
            uint256 executionTimestamp
        )
    {
        primaryCredit = state.primaryCredit[trader];
        secondaryCredit = state.secondaryCredit[trader];
        pendingPrimaryWithdraw = state.pendingPrimaryWithdraw[trader];
        pendingSecondaryWithdraw = state.pendingSecondaryWithdraw[trader];
        executionTimestamp = state.withdrawExecutionTimestamp[trader];
    }

    /**
     * @notice 检查订单发送者是否有效
     * @param orderSender 订单发送者地址
     * @return 是否为有效的订单发送者
     */
    function isOrderSenderValid(address orderSender) external view returns (bool) {
        return state.validOrderSender[orderSender];
    }

    /**
     * @notice 检查快速取款操作员是否有效
     * @param fastWithdrawOperator 操作员地址
     * @return 是否在快速取款白名单中
     */
    function isFastWithdrawalValid(address fastWithdrawOperator) external view returns (bool) {
        return state.fastWithdrawalWhitelist[fastWithdrawOperator];
    }

    /**
     * @notice 查询资金授权额度
     * @param from 授权者地址
     * @param spender 被授权者地址
     * @return primaryCreditAllowed 主资产授权额度
     * @return secondaryCreditAllowed 次级资产授权额度
     */
    function isCreditAllowed(
        address from,
        address spender
    )
        external
        view
        returns (uint256 primaryCreditAllowed, uint256 secondaryCreditAllowed)
    {
        return (state.primaryCreditAllowed[from][spender], state.secondaryCreditAllowed[from][spender]);
    }

    /**
     * @notice 检查操作员是否有效
     * @param client 用户地址
     * @param operator 操作员地址
     * @return 操作员是否被授权
     */
    function isOperatorValid(address client, address operator) external view returns (bool) {
        return state.operatorRegistry[client][operator];
    }

    // ==================== 清算相关查询 ====================

    /**
     * @notice 检查交易者是否安全（维持保证金标准）
     * @param trader 交易者地址
     * @return safe 是否安全（不会被清算）
     * @dev 使用维持保证金率判断，低于此值会被清算
     */
    function isSafe(address trader) external view returns (bool safe) {
        return Liquidation._isMMSafe(state, trader);
    }

    /**
     * @notice 检查交易者是否满足初始保证金要求
     * @param trader 交易者地址
     * @return safe 是否满足初始保证金
     * @dev 初始保证金率 > 维持保证金率
     *      满足初始保证金才能开新仓
     */
    function isIMSafe(address trader) external view returns (bool safe) {
        return Liquidation._isIMSafe(state, trader);
    }

    /**
     * @notice 批量检查交易者是否都安全
     * @param traderList 交易者地址列表
     * @return safe 是否全部安全
     * @dev 交易执行后用于检查所有参与者的安全状态
     */
    function isAllSafe(address[] calldata traderList) external view returns (bool safe) {
        return Liquidation._isAllMMSafe(state, traderList);
    }

    /**
     * @notice 获取永续合约的当前资金费率
     * @param perp 永续合约地址
     * @return 资金费率（累计值，18位小数）
     */
    function getFundingRate(address perp) external view returns (int256) {
        return IPerpetual(perp).getFundingRate();
    }

    /**
     * @notice 获取交易者的风险概况
     * @param trader 交易者地址
     * @return netValue 净值（余额 + 未实现盈亏）
     * @return exposure 敞口（持仓价值的绝对值之和）
     * @return initialMargin 所需初始保证金
     * @return maintenanceMargin 所需维持保证金
     * @dev 核心风控指标：
     *      - 保证金率 = netValue / exposure
     *      - 当保证金率 < maintenanceMargin/exposure 时可被清算
     */
    function getTraderRisk(address trader)
        external
        view
        returns (int256 netValue, uint256 exposure, uint256 initialMargin, uint256 maintenanceMargin)
    {
        (netValue, exposure, initialMargin, maintenanceMargin) = Liquidation.getTotalExposure(state, trader);
    }

    /**
     * @notice 获取清算价格
     * @param trader 交易者地址
     * @param perp 永续合约地址
     * @return liquidationPrice 清算价格
     * @dev 当标记价格达到此价格时，交易者会被清算
     *      返回 0 表示没有清算价格（如空仓或保证金充足）
     */
    function getLiquidationPrice(address trader, address perp) external view returns (uint256 liquidationPrice) {
        return Liquidation.getLiquidationPrice(state, trader, perp);
    }

    /**
     * @notice 查询清算成本（预览）
     * @param perp 永续合约地址
     * @param liquidatedTrader 被清算者地址
     * @param requestPaperAmount 请求清算的数量
     * @return liqtorPaperChange 清算者的 paper 变化
     * @return liqtorCreditChange 清算者的 credit 变化
     * @dev 清算者可以使用此函数预估清算成本
     */
    function getLiquidationCost(
        address perp,
        address liquidatedTrader,
        int256 requestPaperAmount
    )
        external
        view
        returns (int256 liqtorPaperChange, int256 liqtorCreditChange)
    {
        (liqtorPaperChange, liqtorCreditChange,) =
            Liquidation.getLiquidateCreditAmount(state, perp, liquidatedTrader, requestPaperAmount);
    }

    // ==================== 订单相关查询 ====================

    /**
     * @notice 获取订单已成交数量
     * @param orderHash 订单哈希
     * @return filledAmount 已成交的 paper 数量
     * @dev 用于检查订单是否已完全成交或部分成交
     */
    function getOrderFilledAmount(bytes32 orderHash) external view returns (uint256 filledAmount) {
        filledAmount = state.orderFilledPaperAmount[orderHash];
    }

    // ==================== 编码辅助函数 ====================

    /**
     * @notice 编码设置操作员的调用数据
     * @param operator 操作员地址
     * @param isValid 是否有效
     * @return 编码后的调用数据
     * @dev 用于批量调用或子账户操作
     */
    function getSetOperatorCallData(address operator, bool isValid) external pure returns (bytes memory) {
        return abi.encodeWithSignature("setOperator(address,bool)", operator, isValid);
    }

    /**
     * @notice 编码请求取款的调用数据
     * @param from 取款来源
     * @param primaryAmount 主资产数量
     * @param secondaryAmount 次级资产数量
     * @return 编码后的调用数据
     */
    function getRequestWithdrawCallData(
        address from,
        uint256 primaryAmount,
        uint256 secondaryAmount
    )
        external
        pure
        returns (bytes memory)
    {
        return abi.encodeWithSignature("requestWithdraw(address,uint256,uint256)", from, primaryAmount, secondaryAmount);
    }

    /**
     * @notice 编码执行取款的调用数据
     * @param from 取款来源
     * @param to 接收地址
     * @param isInternal 是否内部转账
     * @param param 回调参数
     * @return 编码后的调用数据
     */
    function getExecuteWithdrawCallData(
        address from,
        address to,
        bool isInternal,
        bytes memory param
    )
        external
        pure
        returns (bytes memory)
    {
        return abi.encodeWithSignature("executeWithdraw(address,address,bool,bytes)", from, to, isInternal, param);
    }
}
