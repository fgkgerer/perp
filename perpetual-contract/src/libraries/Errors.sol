/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

/**
 * @title Errors - 错误消息定义
 * @notice 定义系统中所有的错误消息常量
 * @dev 使用常量而非自定义错误类型，以便于调试和日志追踪
 */
library Errors {
    // ==================== 永续合约相关错误 ====================
    
    /// @notice 订单的永续合约地址与调用合约不匹配
    string constant PERP_MISMATCH = "META_PERP_MISMATCH";
    /// @notice 永续合约未注册
    string constant PERP_NOT_REGISTERED = "META_PERP_NOT_REGISTERED";
    /// @notice 永续合约已注册，不能重复注册
    string constant PERP_ALREADY_REGISTERED = "META_PERP_ALREADY_REGISTERED";
    /// @notice 风险参数无效（如清算参数设置不合理）
    string constant INVALID_RISK_PARAM = "META_INVALID_RISK_PARAM";

    // ==================== 订单相关错误 ====================
    
    /// @notice 无效的订单发送者（撮合引擎未授权）
    string constant INVALID_ORDER_SENDER = "META_INVALID_ORDER_SENDER";
    /// @notice 订单签名无效
    string constant INVALID_ORDER_SIGNATURE = "META_INVALID_ORDER_SIGNATURE";
    /// @notice 交易者数量不足（至少需要2个）
    string constant INVALID_TRADER_NUMBER = "META_AT_LEAST_TWO_TRADERS";
    /// @notice 无效的资金费率更新者
    string constant INVALID_FUNDING_RATE_KEEPER = "META_INVALID_FUNDING_RATE_KEEPER";
    /// @notice 无效的清算执行者
    string constant INVALID_LIQUIDATION_EXECUTOR = "META_INVALID_LIQUIDATION_EXECUTOR";
    /// @notice 订单成交数量超过限额
    string constant ORDER_FILLED_OVERFLOW = "META_ORDER_FILLED_OVERFLOW";
    /// @notice 订单价格不匹配（maker 和 taker 价格无法成交）
    string constant ORDER_PRICE_NOT_MATCH = "META_ORDER_PRICE_NOT_MATCH";
    /// @notice 订单价格为负（paper 和 credit 应该异号）
    string constant ORDER_PRICE_NEGATIVE = "META_ORDER_PRICE_NEGATIVE";
    /// @notice 订单发送者账户不安全
    string constant ORDER_SENDER_NOT_SAFE = "META_ORDER_SENDER_NOT_SAFE";
    /// @notice 订单已过期
    string constant ORDER_EXPIRED = "META_ORDER_EXPIRED";
    /// @notice 订单排序错误（maker 订单应按地址升序排列）
    string constant ORDER_WRONG_SORTING = "META_ORDER_WRONG_SORTING";
    /// @notice 禁止自成交（同一签名者的订单不能互相成交）
    string constant ORDER_SELF_MATCH = "META_ORDER_SELF_MATCH";

    // ==================== 账户安全相关错误 ====================
    
    /// @notice 账户不安全（保证金不足）
    string constant ACCOUNT_NOT_SAFE = "META_ACCOUNT_NOT_SAFE";
    /// @notice 账户是安全的（不能清算安全账户）
    string constant ACCOUNT_IS_SAFE = "META_ACCOUNT_IS_SAFE";
    /// @notice Taker 成交数量错误
    string constant TAKER_TRADE_AMOUNT_WRONG = "META_TAKER_TRADE_AMOUNT_WRONG";
    /// @notice 交易者在该市场没有仓位
    string constant TRADER_HAS_NO_POSITION = "META_TRADER_HAS_NO_POSITION";

    // ==================== 取款相关错误 ====================
    
    /// @notice 取款请求在等待中（时间锁未到期）
    string constant WITHDRAW_PENDING = "META_WITHDRAW_PENDING";
    /// @notice 取款请求无效（权限或金额问题）
    string constant WITHDRAW_INVALID = "META_WITHDRAW_INVALID";
    /// @notice 快速取款不被允许
    string constant FAST_WITHDRAW_NOT_ALLOWED = "META_FAST_WITHDRAW_NOT_ALLOWED";

    // ==================== 清算相关错误 ====================
    
    /// @notice 清算请求数量错误（方向不对）
    string constant LIQUIDATION_REQUEST_AMOUNT_WRONG = "META_LIQUIDATION_REQUEST_AMOUNT_WRONG";
    /// @notice 不允许自我清算
    string constant SELF_LIQUIDATION_NOT_ALLOWED = "META_SELF_LIQUIDATION_NOT_ALLOWED";

    // ==================== 资产相关错误 ====================
    
    /// @notice 次级资产已存在，不能重复设置
    string constant SECONDARY_ASSET_ALREADY_EXIST = "META_SECONDARY_ASSET_ALREADY_EXIST";
    /// @notice 次级资产小数位数与主资产不匹配
    string constant SECONDARY_ASSET_DECIMAL_WRONG = "META_SECONDARY_ASSET_DECIMAL_WRONG";
    /// @notice 数组长度不匹配
    string constant ARRAY_LENGTH_NOT_SAME = "META_ARRAY_LENGTH_NOT_SAME";
    /// @notice 持仓数量达到上限
    string constant POSITION_AMOUNT_REACH_UPPER_LIMIT = "META_POSITION_AMOUNT_REACH_UPPER_LIMIT";

    // ==================== 借贷系统相关错误 ====================
    
    /// @notice 储备不允许存款
    string constant RESERVE_NOT_ALLOW_DEPOSIT = "RESERVE_NOT_ALLOW_DEPOSIT";
    /// @notice 存款金额为零
    string constant DEPOSIT_AMOUNT_IS_ZERO = "DEPOSIT_AMOUNT_IS_ZERO";
    /// @notice 还款金额为零
    string constant REPAY_AMOUNT_IS_ZERO = "REPAY_AMOUNT_IS_ZERO";
    /// @notice 取款金额为零
    string constant WITHDRAW_AMOUNT_IS_ZERO = "WITHDRAW_AMOUNT_IS_ZERO";
    /// @notice 清算金额为零
    string constant LIQUIDATE_AMOUNT_IS_ZERO = "LIQUIDATE_AMOUNT_IS_ZERO";
    /// @notice 借款后账户不安全
    string constant AFTER_BORROW_ACCOUNT_IS_NOT_SAFE = "AFTER_BORROW_ACCOUNT_IS_NOT_SAFE";
    /// @notice 取款后账户不安全
    string constant AFTER_WITHDRAW_ACCOUNT_IS_NOT_SAFE = "AFTER_WITHDRAW_ACCOUNT_IS_NOT_SAFE";
    /// @notice 闪电贷后账户不安全
    string constant AFTER_FLASHLOAN_ACCOUNT_IS_NOT_SAFE = "AFTER_FLASHLOAN_ACCOUNT_IS_NOT_SAFE";
    /// @notice 单账户存款超限
    string constant EXCEED_THE_MAX_DEPOSIT_AMOUNT_PER_ACCOUNT = "EXCEED_THE_MAX_DEPOSIT_AMOUNT_PER_ACCOUNT";
    /// @notice 总存款超限
    string constant EXCEED_THE_MAX_DEPOSIT_AMOUNT_TOTAL = "EXCEED_THE_MAX_DEPOSIT_AMOUNT_TOTAL";
    /// @notice 单账户借款超限
    string constant EXCEED_THE_MAX_BORROW_AMOUNT_PER_ACCOUNT = "EXCEED_THE_MAX_BORROW_AMOUNT_PER_ACCOUNT";
    /// @notice 总借款超限
    string constant EXCEED_THE_MAX_BORROW_AMOUNT_TOTAL = "EXCEED_THE_MAX_BORROW_AMOUNT_TOTAL";
    /// @notice 取款金额过大
    string constant WITHDRAW_AMOUNT_IS_TOO_BIG = "WITHDRAW_AMOUNT_IS_TOO_BIG";
    /// @notice 无法操作该账户（权限不足）
    string constant CAN_NOT_OPERATE_ACCOUNT = "CAN_NOT_OPERATE_ACCOUNT";
    /// @notice 清算价格保护触发
    string constant LIQUIDATION_PRICE_PROTECTION = "LIQUIDATION_PRICE_PROTECTION";
    /// @notice 不允许兑换
    string constant NOT_ALLOWED_TO_EXCHANGE = "NOT_ALLOWED_TO_EXCHANGE";
    /// @notice 不允许添加更多储备
    string constant NO_MORE_RESERVE_ALLOWED = "NO_MORE_RESERVE_ALLOWED";
    /// @notice 储备参数错误
    string constant RESERVE_PARAM_ERROR = "RESERVE_PARAM_ERROR";
    /// @notice 还款金额不足
    string constant REPAY_AMOUNT_NOT_ENOUGH = "REPAY_AMOUNT_NOT_ENOUGH";
    /// @notice 保险金额不足
    string constant INSURANCE_AMOUNT_NOT_ENOUGH = "INSURANCE_AMOUNT_NOT_ENOUGH";
    /// @notice 清算金额不足
    string constant LIQUIDATED_AMOUNT_NOT_ENOUGH = "LIQUIDATED_AMOUNT_NOT_ENOUGH";
    /// @notice 清算者不在白名单中
    string constant LIQUIDATOR_NOT_IN_THE_WHITELIST = "LIQUIDATOR_NOT_IN_THE_WHITELIST";
    /// @notice 储备参数错误
    string constant RESERVE_PARAM_WRONG = "RESERVE_PARAM_WRONG";
}
