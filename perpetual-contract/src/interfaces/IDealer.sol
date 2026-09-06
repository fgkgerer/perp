/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

import "../libraries/Types.sol";

/**
 * @title IDealer - 交易系统接口
 * @notice MetaNodeDealer 合约的接口定义
 * 
 * 功能分类：
 * 1. 资金管理：存款、取款、内部转账
 * 2. 交易核心：订单验证、结算
 * 3. 风险管理：安全检查、清算
 * 4. 信息查询：账户状态、市场参数
 */
interface IDealer {
    // ==================== 资金管理 ====================

    /**
     * @notice 存入保证金
     * @param primaryAmount 主资产存款数量
     * @param secondaryAmount 次级资产存款数量
     * @param to 存入的目标账户地址
     * @dev 调用方需提前 approve 足够额度
     */
    function deposit(uint256 primaryAmount, uint256 secondaryAmount, address to) external;

    /**
     * @notice 请求取款（第一步）
     * @param from 取款来源账户
     * @param primaryAmount 主资产取款数量
     * @param secondaryAmount 次级资产取款数量
     * @dev 主要目的是避免因取款导致对手方交易失败
     *      取款请求需要等待时间锁后才能执行
     */
    function requestWithdraw(address from, uint256 primaryAmount, uint256 secondaryAmount) external;

    /**
     * @notice 执行取款（第二步）
     * @param from 取款来源账户
     * @param to 资金接收地址
     * @param isInternal 是否仅内部转账（不实际转出 ERC20）
     * @param param 回调参数，非空时会调用 to 地址
     */
    function executeWithdraw(address from, address to, bool isInternal, bytes memory param) external;

    /**
     * @notice 快速取款（一步完成）
     * @param from 取款来源账户
     * @param to 资金接收地址
     * @param primaryAmount 主资产数量
     * @param secondaryAmount 次级资产数量
     * @param isInternal 是否仅内部转账
     * @param param 回调参数
     * @dev 跳过时间锁直接取款，需要特殊权限
     */
    function fastWithdraw(
        address from,
        address to,
        uint256 primaryAmount,
        uint256 secondaryAmount,
        bool isInternal,
        bytes memory param
    )
        external;

    // ==================== 交易核心 ====================

    /**
     * @notice 批准交易（核心撮合函数）
     * @param orderSender 订单发送者（撮合引擎）地址
     * @param tradeData 包含订单、签名和撮合信息的编码数据
     * @return traderList 参与交易的交易者列表
     * @return paperChangeList 各交易者的 paper 变化
     * @return creditChangeList 各交易者的 credit 变化
     * @dev 仅永续合约可调用此函数
     *      解析 tradeData，验证订单并返回各方余额变化
     */
    function approveTrade(
        address orderSender,
        bytes calldata tradeData
    )
        external
        returns (address[] memory traderList, int256[] memory paperChangeList, int256[] memory creditChangeList);

    // ==================== 风险管理 ====================

    /**
     * @notice 检查交易者是否安全（满足维持保证金）
     * @param trader 交易者地址
     * @return 是否安全（不会被清算）
     * @dev 如果不安全，该交易者所有市场的仓位都可能被清算
     */
    function isSafe(address trader) external view returns (bool);

    /**
     * @notice 批量检查交易者安全状态
     * @param traderList 交易者列表
     * @return 是否全部安全
     * @dev 通过缓存标记价格提高 gas 效率
     */
    function isAllSafe(address[] calldata traderList) external view returns (bool);

    /**
     * @notice 请求清算
     * @param executor 执行清算的地址
     * @param liquidator 清算者（接手仓位）
     * @param liquidatedTrader 被清算者
     * @param requestPaperAmount 请求清算的仓位数量
     *        正数表示清算多头仓位，负数表示清算空头仓位
     * @return liqtorPaperChange 清算者的 paper 变化
     * @return liqtorCreditChange 清算者的 credit 变化
     * @return liqedPaperChange 被清算者的 paper 变化
     * @return liqedCreditChange 被清算者的 credit 变化
     * @dev 仅永续合约可调用
     *      liqtor = liquidator, liqed = liquidated trader
     */
    function requestLiquidation(
        address executor,
        address liquidator,
        address liquidatedTrader,
        int256 requestPaperAmount
    )
        external
        returns (int256 liqtorPaperChange, int256 liqtorCreditChange, int256 liqedPaperChange, int256 liqedCreditChange);

    /**
     * @notice 处理坏账
     * @param liquidatedTrader 被清算者地址
     * @dev 将所有坏账（包括主资产和次级资产）转移给保险账户
     */
    function handleBadDebt(address liquidatedTrader) external;

    // ==================== 仓位管理 ====================

    /**
     * @notice 注册开仓
     * @param trader 交易者地址
     * @dev 仅永续合约可调用
     *      当交易者开仓时由 Perpetual 合约调用
     */
    function openPosition(address trader) external;

    /**
     * @notice 实现盈亏并移除仓位
     * @param trader 交易者地址
     * @param pnl 盈亏金额
     * @dev 仅永续合约可调用
     *      当交易者平仓时由 Perpetual 合约调用
     */
    function realizePnl(address trader, int256 pnl) external;

    // ==================== 权限管理 ====================

    /**
     * @notice 设置操作员
     * @param operator 操作员地址
     * @param isValid 是否授权
     * @dev 操作员可以代替用户签名订单
     */
    function setOperator(address operator, bool isValid) external;

    // ==================== 查询函数 ====================

    /**
     * @notice 获取永续合约的风险参数
     * @param perp 永续合约地址
     * @return params 风险参数结构体
     */
    function getRiskParams(address perp) external view returns (Types.RiskParams memory params);

    /**
     * @notice 获取所有已注册的永续合约市场
     * @return 永续合约地址数组
     */
    function getAllRegisteredPerps() external view returns (address[] memory);

    /**
     * @notice 获取标记价格
     * @param perp 永续合约地址
     * @return 标记价格（1e18 精度）
     */
    function getMarkPrice(address perp) external view returns (uint256);

    /**
     * @notice 获取交易者的所有持仓市场
     * @param trader 交易者地址
     * @return 持仓的永续合约地址数组
     */
    function getPositions(address trader) external view returns (address[] memory);

    /**
     * @notice 获取交易者的资金详情
     * @param trader 交易者地址
     * @return primaryCredit 主资产余额
     * @return secondaryCredit 次级资产余额
     * @return pendingPrimaryWithdraw 待取款主资产
     * @return pendingSecondaryWithdraw 待取款次级资产
     * @return executionTimestamp 取款可执行时间
     * @dev 注意：不能直接将 credit 作为净值或保证金
     *      需要结合仓位价值一起计算
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
        );

    /**
     * @notice 获取交易者的风险概况
     * @param trader 交易者地址
     * @return netValue 净值（包含 credit 和仓位价值）
     * @return exposure 敞口（所有市场仓位价值绝对值之和）
     * @return initialMargin 初始保证金要求
     * @return maintenanceMargin 维持保证金要求
     */
    function getTraderRisk(address trader)
        external
        view
        returns (int256 netValue, uint256 exposure, uint256 initialMargin, uint256 maintenanceMargin);

    /**
     * @notice 获取清算价格
     * @param trader 交易者地址
     * @param perp 永续合约地址
     * @return liquidationPrice 清算价格，0 表示无清算价格
     * @dev 此函数用于参考，误差通常在 10 wei 以内
     */
    function getLiquidationPrice(address trader, address perp) external view returns (uint256 liquidationPrice);

    /**
     * @notice 预览清算成本
     * @param perp 永续合约地址
     * @param liquidatedTrader 被清算者
     * @param requestPaperAmount 请求清算数量
     * @return liqtorPaperChange 清算者的 paper 变化
     * @return liqtorCreditChange 清算者的 credit 变化
     * @dev 清算者可用此函数提前计算需要支付的金额
     */
    function getLiquidationCost(
        address perp,
        address liquidatedTrader,
        int256 requestPaperAmount
    )
        external
        view
        returns (int256 liqtorPaperChange, int256 liqtorCreditChange);

    /**
     * @notice 获取订单已成交数量
     * @param orderHash 订单哈希
     * @return filledAmount 已成交的 paper 数量
     * @dev 用于避免重复撮合
     */
    function getOrderFilledAmount(bytes32 orderHash) external view returns (uint256 filledAmount);

    /**
     * @notice 获取资金费率
     * @param perp 永续合约地址
     * @return 资金费率（1e18 精度）
     */
    function getFundingRate(address perp) external view returns (int256);

    /**
     * @notice 批量更新资金费率
     * @param perpList 永续合约地址列表
     * @param rateList 资金费率列表
     * @dev 仅资金费率更新者可调用
     */
    function updateFundingRate(address[] calldata perpList, int256[] calldata rateList) external;

    // ==================== 权限检查 ====================

    /**
     * @notice 检查订单发送者是否有效
     * @param orderSender 订单发送者地址
     * @return 是否为有效的订单发送者
     */
    function isOrderSenderValid(address orderSender) external view returns (bool);

    /**
     * @notice 检查快速取款操作员是否有效
     * @param fastWithdrawOperator 操作员地址
     * @return 是否有效
     */
    function isFastWithdrawalValid(address fastWithdrawOperator) external view returns (bool);

    /**
     * @notice 检查操作员是否有效
     * @param client 用户地址
     * @param operator 操作员地址
     * @return 操作员是否被授权
     */
    function isOperatorValid(address client, address operator) external view returns (bool);

    /**
     * @notice 查询资金操作授权额度
     * @param from 授权者
     * @param spender 被授权者
     * @return primaryCreditAllowed 主资产授权额度
     * @return secondaryCreditAllowed 次级资产授权额度
     */
    function isCreditAllowed(
        address from,
        address spender
    )
        external
        view
        returns (uint256 primaryCreditAllowed, uint256 secondaryCreditAllowed);
}
