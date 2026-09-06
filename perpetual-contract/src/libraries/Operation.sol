/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

import "../interfaces/IPerpetual.sol";
import "../interfaces/internal/IDecimalERC20.sol";
import "../libraries/Errors.sol";
import "./Types.sol";

/**
 * @title Operation - 操作管理库
 * @notice 处理系统配置和权限管理
 * 
 * 主要功能：
 * 1. 市场管理：注册/注销永续合约市场
 * 2. 资金费率：更新各市场的资金费率
 * 3. 系统参数：设置保险账户、时间锁等
 * 4. 权限管理：设置操作员、白名单等
 */
library Operation {
    // ==================== 事件 ====================

    /// @notice 资金费率更新者变更事件
    event SetFundingRateKeeper(address oldKeeper, address newKeeper);

    /// @notice 保险账户变更事件
    event SetInsurance(address oldInsurance, address newInsurance);

    /// @notice 最大持仓数量变更事件
    event SetMaxPositionAmount(uint256 oldMaxPositionAmount, uint256 newMaxPositionAmount);

    /// @notice 取款时间锁变更事件
    event SetWithdrawTimeLock(uint256 oldWithdrawTimeLock, uint256 newWithdrawTimeLock);

    /// @notice 订单发送者设置事件
    event SetOrderSender(address orderSender, bool isValid);

    /// @notice 快速取款白名单设置事件
    event SetFastWithdrawalWhitelist(address target, bool isValid);

    /// @notice 取款白名单设置事件
    event SetWithdrawalWhitelist(address target, bool isValid);

    /// @notice 快速取款禁用状态变更事件
    event FastWithdrawDisabled(bool disabled);

    /// @notice 操作员设置事件
    /// @param client 用户地址
    /// @param operator 操作员地址
    /// @param isValid 是否有效
    event SetOperator(address indexed client, address indexed operator, bool isValid);

    /// @notice 资金操作员授权额度变更事件
    /// @param client 用户地址
    /// @param operator 操作员地址
    /// @param primaryAllowed 主资产授权额度
    /// @param secondaryAllowed 次级资产授权额度
    event FundOperatorAllowedChange(
        address indexed client, address indexed operator, uint256 primaryAllowed, uint256 secondaryAllowed
    );

    /// @notice 次级资产设置事件
    event SetSecondaryAsset(address secondaryAsset);

    /// @notice 风险参数更新事件
    event UpdatePerpRiskParams(address indexed perp, Types.RiskParams param);

    /// @notice 资金费率更新事件
    event UpdateFundingRate(address indexed perp, int256 oldRate, int256 newRate);

    // ==================== 函数 ====================

    /**
     * @notice 设置永续合约的风险参数
     * @param state 系统状态
     * @param perp 永续合约地址
     * @param param 风险参数
     * @dev 功能：
     *      1. 注册新市场（param.isRegistered = true）
     *      2. 注销市场（param.isRegistered = false）
     *      3. 更新已有市场参数
     * 
     * 参数验证：
     * - liquidationPriceOff + insuranceFeeRate <= liquidationThreshold
     *   确保清算折扣和保险费之和不超过清算阈值
     */
    function setPerpRiskParams(Types.State storage state, address perp, Types.RiskParams calldata param) external {
        // 如果从注册状态变为未注册，从列表中移除
        if (state.perpRiskParams[perp].isRegistered && !param.isRegistered) {
            for (uint256 i; i < state.registeredPerp.length;) {
                if (state.registeredPerp[i] == perp) {
                    state.registeredPerp[i] = state.registeredPerp[state.registeredPerp.length - 1];
                    state.registeredPerp.pop();
                }
                unchecked {
                    ++i;
                }
            }
        }
        // 如果是新注册的市场，添加到列表
        if (!state.perpRiskParams[perp].isRegistered && param.isRegistered) {
            state.registeredPerp.push(perp);
        }
        // 验证参数合理性
        require(
            param.liquidationPriceOff + param.insuranceFeeRate <= param.liquidationThreshold, Errors.INVALID_RISK_PARAM
        );
        // 保存参数
        state.perpRiskParams[perp] = param;
        emit UpdatePerpRiskParams(perp, param);
    }

    /**
     * @notice 批量更新资金费率
     * @param perpList 永续合约地址列表
     * @param rateList 新资金费率列表
     * @dev 资金费率机制说明：
     *      - 正费率：多头支付给空头
     *      - 负费率：空头支付给多头
     *      - 费率是累计值，变化量才是实际支付
     * 
     * 由 fundingRateKeeper（通常是链下服务）定期调用
     */
    function updateFundingRate(address[] calldata perpList, int256[] calldata rateList) external {
        require(perpList.length == rateList.length, Errors.ARRAY_LENGTH_NOT_SAME);
        for (uint256 i = 0; i < perpList.length;) {
            int256 oldRate = IPerpetual(perpList[i]).getFundingRate();
            IPerpetual(perpList[i]).updateFundingRate(rateList[i]);
            emit UpdateFundingRate(perpList[i], oldRate, rateList[i]);
            unchecked {
                ++i;
            }
        }
    }

    /**
     * @notice 设置资金费率更新者
     * @param state 系统状态
     * @param newKeeper 新的更新者地址
     */
    function setFundingRateKeeper(Types.State storage state, address newKeeper) external {
        address oldKeeper = state.fundingRateKeeper;
        state.fundingRateKeeper = newKeeper;
        emit SetFundingRateKeeper(oldKeeper, newKeeper);
    }

    /**
     * @notice 设置保险账户
     * @param state 系统状态
     * @param newInsurance 新的保险账户地址
     * @dev 保险账户功能：
     *      1. 收取清算时的保险费
     *      2. 承担交易者的坏账
     */
    function setInsurance(Types.State storage state, address newInsurance) external {
        address oldInsurance = state.insurance;
        state.insurance = newInsurance;
        emit SetInsurance(oldInsurance, newInsurance);
    }

    /**
     * @notice 设置单用户最大持仓市场数量
     * @param state 系统状态
     * @param newMaxPositionAmount 新的最大数量
     * @dev 限制持仓数量的原因：
     *      计算风险时需要遍历所有持仓市场
     *      数量过多会导致 gas 消耗过大
     */
    function setMaxPositionAmount(Types.State storage state, uint256 newMaxPositionAmount) external {
        uint256 oldMaxPositionAmount = state.maxPositionAmount;
        state.maxPositionAmount = newMaxPositionAmount;
        emit SetMaxPositionAmount(oldMaxPositionAmount, newMaxPositionAmount);
    }

    /**
     * @notice 设置取款时间锁
     * @param state 系统状态
     * @param newWithdrawTimeLock 新的时间锁（秒）
     * @dev 时间锁的作用：
     *      防止用户在提交订单后、成交前取走保证金
     *      给链下撮合系统足够的时间处理订单
     */
    function setWithdrawTimeLock(Types.State storage state, uint256 newWithdrawTimeLock) external {
        uint256 oldWithdrawTimeLock = state.withdrawTimeLock;
        state.withdrawTimeLock = newWithdrawTimeLock;
        emit SetWithdrawTimeLock(oldWithdrawTimeLock, newWithdrawTimeLock);
    }

    /**
     * @notice 设置订单发送者（撮合引擎）
     * @param state 系统状态
     * @param orderSender 订单发送者地址
     * @param isValid 是否有效
     * @dev 只有授权的订单发送者才能提交撮合交易
     *      通常是链下撮合引擎的地址
     */
    function setOrderSender(Types.State storage state, address orderSender, bool isValid) external {
        state.validOrderSender[orderSender] = isValid;
        emit SetOrderSender(orderSender, isValid);
    }

    /**
     * @notice 设置快速取款白名单
     * @param state 系统状态
     * @param target 目标地址
     * @param isValid 是否在白名单中
     * @dev 白名单用户可以跳过取款时间锁
     */
    function setFastWithdrawalWhitelist(Types.State storage state, address target, bool isValid) external {
        state.fastWithdrawalWhitelist[target] = isValid;
        emit SetFastWithdrawalWhitelist(target, isValid);
    }

    /**
     * @notice 禁用/启用快速取款
     * @param state 系统状态
     * @param disabled 是否禁用
     * @dev 紧急情况下可以全局禁用快速取款
     */
    function disableFastWithdraw(Types.State storage state, bool disabled) external {
        state.fastWithdrawDisabled = disabled;
        emit FastWithdrawDisabled(disabled);
    }

    /**
     * @notice 设置取款白名单
     * @param state 系统状态
     * @param target 目标地址
     * @param isValid 是否在白名单中
     * @dev 白名单地址可以接收取款回调
     */
    function setWithdrawalWhitelist(Types.State storage state, address target, bool isValid) external {
        state.isWithdrawalWhitelist[target] = isValid;
        emit SetFastWithdrawalWhitelist(target, isValid);
    }

    /**
     * @notice 设置操作员
     * @param state 系统状态
     * @param client 用户地址
     * @param operator 操作员地址
     * @param isValid 是否有效
     * @dev 操作员权限：
     *      1. 代替用户签名订单
     *      2. 执行用户的清算
     *      不包括资金操作权限
     */
    function setOperator(Types.State storage state, address client, address operator, bool isValid) external {
        state.operatorRegistry[client][operator] = isValid;
        emit SetOperator(client, operator, isValid);
    }

    /**
     * @notice 授权资金操作员
     * @param state 系统状态
     * @param client 用户地址
     * @param operator 操作员地址
     * @param primaryAmount 主资产授权额度
     * @param secondaryAmount 次级资产授权额度
     * @dev 资金操作员可以在授权额度内代替用户取款
     */
    function approveFundOperator(
        Types.State storage state,
        address client,
        address operator,
        uint256 primaryAmount,
        uint256 secondaryAmount
    )
        external
    {
        state.primaryCreditAllowed[client][operator] = primaryAmount;
        state.secondaryCreditAllowed[client][operator] = secondaryAmount;
        emit FundOperatorAllowedChange(client, operator, primaryAmount, secondaryAmount);
    }

    /**
     * @notice 设置次级资产
     * @param state 系统状态
     * @param _secondaryAsset 次级资产地址
     * @dev 限制：
     *      1. 只能设置一次
     *      2. 小数位数必须与主资产相同
     */
    function setSecondaryAsset(Types.State storage state, address _secondaryAsset) external {
        require(state.secondaryAsset == address(0), Errors.SECONDARY_ASSET_ALREADY_EXIST);
        require(
            IDecimalERC20(_secondaryAsset).decimals() == IDecimalERC20(state.primaryAsset).decimals(),
            Errors.SECONDARY_ASSET_DECIMAL_WRONG
        );
        state.secondaryAsset = _secondaryAsset;
        emit SetSecondaryAsset(_secondaryAsset);
    }
}
