/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

/**
 * @title Types - 系统数据类型定义
 * @notice 定义永续合约系统使用的所有核心数据结构
 */
library Types {
    /// @notice 每年的秒数，用于利率计算
    uint256 public constant SECONDS_PER_YEAR = 365 days;

    /// @notice 基数单位 1e18，用于精度计算
    uint256 public constant ONE = 1e18;

    /**
     * @notice Dealer 合约的核心状态结构
     * @dev 包含所有用户余额、市场参数、订单记录等关键数据
     */
    struct State {
        /// @notice 主资产地址（通常是 USDC），用于保证金和结算
        address primaryAsset;
        /// @notice 次级资产地址（可选），支持多种保证金类型
        address secondaryAsset;

        // ========== 用户余额相关 ==========
        /// @notice 用户主资产余额（可以为负，表示亏损）
        mapping(address => int256) primaryCredit;
        /// @notice 用户次级资产余额
        mapping(address => uint256) secondaryCredit;
        /// @notice 授权资金操作员的主资产额度：用户 => 操作员 => 额度
        mapping(address => mapping(address => uint256)) primaryCreditAllowed;
        /// @notice 授权资金操作员的次级资产额度
        mapping(address => mapping(address => uint256)) secondaryCreditAllowed;

        // ========== 取款相关 ==========
        /// @notice 取款时间锁（秒），防止在订单成交前取走保证金
        uint256 withdrawTimeLock;
        /// @notice 用户待取款的主资产数量
        mapping(address => uint256) pendingPrimaryWithdraw;
        /// @notice 用户待取款的次级资产数量
        mapping(address => uint256) pendingSecondaryWithdraw;
        /// @notice 用户取款可执行的时间戳
        mapping(address => uint256) withdrawExecutionTimestamp;

        // ========== 市场相关 ==========
        /// @notice 永续合约的风险参数：合约地址 => 参数
        mapping(address => Types.RiskParams) perpRiskParams;
        /// @notice 所有已注册的永续合约地址列表
        address[] registeredPerp;

        // ========== 持仓相关 ==========
        /// @notice 用户的持仓市场列表：用户 => 永续合约地址数组
        mapping(address => address[]) openPositions;
        /// @notice 仓位序列号，用于链下盈亏计算，每次全平仓+1
        /// 用户 => 永续合约 => 序列号
        mapping(address => mapping(address => uint256)) positionSerialNum;

        // ========== 订单相关 ==========
        /// @notice 订单已成交数量：订单哈希 => 已成交 paper 数量
        mapping(bytes32 => uint256) orderFilledPaperAmount;
        /// @notice 有效的订单发送者（撮合引擎）
        mapping(address => bool) validOrderSender;

        // ========== 权限和白名单 ==========
        /// @notice 快速取款白名单
        mapping(address => bool) fastWithdrawalWhitelist;
        /// @notice 是否禁用快速取款
        bool fastWithdrawDisabled;
        /// @notice 操作员注册表：用户 => 操作员 => 是否有效
        mapping(address => mapping(address => bool)) operatorRegistry;
        /// @notice 取款白名单
        mapping(address => bool) isWithdrawalWhitelist;

        // ========== 系统账户 ==========
        /// @notice 保险账户地址，用于收取保险费和承担坏账
        address insurance;
        /// @notice 资金费率更新者地址
        address fundingRateKeeper;
        /// @notice 单用户最大持仓市场数量
        uint256 maxPositionAmount;
    }

    /**
     * @notice 订单结构体
     * @dev 用于链下签名、链上验证的订单格式
     * 
     * 订单方向说明：
     * - 做多（买入）：paperAmount > 0, creditAmount < 0
     * - 做空（卖出）：paperAmount < 0, creditAmount > 0
     * 
     * 价格计算：price = |creditAmount| / |paperAmount|
     */
    struct Order {
        /// @notice 目标永续合约地址
        address perp;
        /**
         * @notice 订单签名者地址（交易者身份）
         * @dev 签名者的余额会因交易而变化
         *      - 通常是 EOA 地址，需要本人签名
         *      - 如果是合约地址，可以：
         *        1. 通过 setOperator 授权其他 EOA 签名
         *        2. 实现 IERC1271 接口自验证
         */
        address signer;
        /// @notice paper 数量（仓位），正数做多，负数做空
        int128 paperAmount;
        /// @notice credit 数量（资金），与 paperAmount 异号
        int128 creditAmount;
        /**
         * @notice 订单附加信息，打包了多个字段
         * 
         * ╔═══════════════════╤═════════╗
         * ║ 字段              │ 类型    ║
         * ╟───────────────────┼─────────╢
         * ║ makerFeeRate      │ int64   ║  费率（负数表示返佣）
         * ║ takerFeeRate      │ int64   ║  费率
         * ║ expiration        │ uint64  ║  过期时间戳
         * ║ nonce             │ uint64  ║  随机数（防重放）
         * ╚═══════════════════╧═════════╝
         */
        bytes32 info;
    }

    /// @notice EIP-712 订单类型哈希，用于签名验证
    bytes32 public constant ORDER_TYPEHASH =
        keccak256("Order(address perp,address signer,int128 paperAmount,int128 creditAmount,bytes32 info)");

    /**
     * @notice 永续合约风险参数
     * @dev 控制交易的杠杆和清算行为
     */
    struct RiskParams {
        /**
         * @notice 初始保证金率
         * @dev 开仓和取款时需要满足：netValue >= exposure * initialMarginRatio
         *      比如 10% 意味着最大 10 倍杠杆
         *      1e18 基数小数
         */
        uint256 initialMarginRatio;
        /**
         * @notice 维持保证金率（清算阈值）
         * @dev 当 netValue < exposure * liquidationThreshold 时会被清算
         *      值越低，允许的杠杆越高
         *      1e18 基数小数
         */
        uint256 liquidationThreshold;
        /**
         * @notice 清算价格折扣
         * @dev 清算时的价格优惠：
         *      - 清算多头：markPrice * (1 - liquidationPriceOff)
         *      - 清算空头：markPrice * (1 + liquidationPriceOff)
         *      1e18 基数小数
         */
        uint256 liquidationPriceOff;
        /// @notice 清算保险费率，从被清算者收取，转入保险账户
        uint256 insuranceFeeRate;
        /// @notice 标记价格来源合约地址
        address markPriceSource;
        /// @notice 市场名称（如 "BTC-PERP"）
        string name;
        /// @notice 是否已注册激活（true 才能交易）
        bool isRegistered;
    }

    /**
     * @notice 订单撮合结果
     * @dev 包含所有参与交易者的余额变化
     */
    struct MatchResult {
        /// @notice 参与交易的交易者列表（第一个是 taker，其余是 maker）
        address[] traderList;
        /// @notice 各交易者的 paper 变化
        int256[] paperChangeList;
        /// @notice 各交易者的 credit 变化（已扣除手续费）
        int256[] creditChangeList;
        /// @notice 订单发送者收取的总手续费
        int256 orderSenderFee;
    }

    /**
     * @notice 抵押品信息（用于借贷系统）
     * @dev 注意：本教学项目主要关注永续合约，此结构体为扩展用途
     */
    struct ReserveInfo {
        /// @notice 初始抵押率，1e18 基数
        uint256 initialMortgageRate;
        /// @notice 最大总存款量
        uint256 maxTotalDepositAmount;
        /// @notice 单账户最大存款量
        uint256 maxDepositAmountPerAccount;
        /// @notice 单账户最大借款量
        uint256 maxColBorrowPerAccount;
        /// @notice 价格预言机地址
        address oracle;
        /// @notice 当前总存款量
        uint256 totalDepositAmount;
        /// @notice 清算抵押率
        uint256 liquidationMortgageRate;
        /// @notice 清算价格折扣
        uint256 liquidationPriceOff;
        /// @notice 保险费率
        uint256 insuranceFeeRate;
        /// @notice 是否最终清算状态（禁止存款和借款）
        bool isFinalLiquidation;
        /// @notice 是否允许存款
        bool isDepositAllowed;
        /// @notice 是否允许借款
        bool isBorrowAllowed;
    }

    /**
     * @notice 用户信息（用于借贷系统）
     */
    struct UserInfo {
        /// @notice 抵押品存款余额：抵押品地址 => 数量
        mapping(address => uint256) depositBalance;
        /// @notice 是否有该抵押品
        mapping(address => bool) hasCollateral;
        /// @notice T0 时刻的借款余额
        uint256 t0BorrowBalance;
        /// @notice 用户的抵押品列表
        address[] collateralList;
    }

    /**
     * @notice 清算数据
     */
    struct LiquidateData {
        /// @notice 实际清算的抵押品数量
        uint256 actualCollateral;
        /// @notice 保险费
        uint256 insuranceFee;
        /// @notice 实际清算的 T0 债务
        uint256 actualLiquidatedT0;
        /// @notice 实际清算的债务
        uint256 actualLiquidated;
        /// @notice 清算后剩余的 USDC
        uint256 liquidatedRemainUSDC;
    }
}
