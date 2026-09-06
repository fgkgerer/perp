/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import "./interfaces/IDealer.sol";
import "./libraries/Errors.sol";
import "./libraries/Operation.sol";
import "./libraries/Types.sol";
import "./MetaNodeStorage.sol";

/**
 * @title MetaNodeOperation - 永续合约系统管理函数
 * @notice 包含仅限管理员调用的系统配置函数
 * 
 * 主要功能：
 * 1. 市场管理：注册永续合约市场、设置风险参数
 * 2. 系统配置：设置保险账户、取款时间锁、最大仓位等
 * 3. 权限管理：设置订单发送者、快速取款白名单
 * 4. 资金费率：更新各市场的资金费率
 */
abstract contract MetaNodeOperation is MetaNodeStorage, IDealer {
    using SafeERC20 for IERC20;

    // ==================== 参数更新函数 ====================

    /**
     * @notice 更新资金费率（批量）
     * @param perpList 永续合约地址列表
     * @param rateList 对应的新资金费率列表
     * @dev 仅 fundingRateKeeper 可调用
     *      资金费率机制：使永续合约价格趋近现货价格
     *      - 正资金费率：多头支付空头
     *      - 负资金费率：空头支付多头
     */
    function updateFundingRate(
        address[] calldata perpList,
        int256[] calldata rateList
    )
        external
        onlyFundingRateKeeper
    {
        Operation.updateFundingRate(perpList, rateList);
    }

    /**
     * @notice 设置永续合约的风险参数
     * @param perp 永续合约地址
     * @param param 风险参数结构体
     * @dev 风险参数包括：
     *      - initialMarginRatio: 初始保证金率（开仓时需要的保证金比例）
     *      - maintenanceMarginRatio: 维持保证金率（低于此比例会被清算）
     *      - liquidationPriceOff: 清算价格折扣
     *      - insuranceFeeRate: 保险费率
     *      - markPriceSource: 标记价格来源
     *      - isRegistered: 是否已注册（true 才能交易）
     */
    function setPerpRiskParams(address perp, Types.RiskParams calldata param) external onlyOwner {
        Operation.setPerpRiskParams(state, perp, param);
    }

    /**
     * @notice 设置资金费率更新者地址
     * @param newKeeper 新的更新者地址
     * @dev 资金费率通常由链下服务定期计算并更新
     */
    function setFundingRateKeeper(address newKeeper) external onlyOwner {
        Operation.setFundingRateKeeper(state, newKeeper);
    }

    /**
     * @notice 设置保险账户地址
     * @param newInsurance 新的保险账户地址
     * @dev 保险账户用于：
     *      1. 收取清算产生的保险费
     *      2. 承担坏账损失
     */
    function setInsurance(address newInsurance) external onlyOwner {
        Operation.setInsurance(state, newInsurance);
    }

    /**
     * @notice 设置最大持仓数量
     * @param newMaxPositionAmount 新的最大持仓数
     * @dev 限制单个账户同时持有的市场数量，防止 gas 消耗过大
     */
    function setMaxPositionAmount(uint256 newMaxPositionAmount) external onlyOwner {
        Operation.setMaxPositionAmount(state, newMaxPositionAmount);
    }

    /**
     * @notice 设置取款时间锁
     * @param newWithdrawTimeLock 新的时间锁（秒）
     * @dev 取款需要等待的时间，防止在订单成交前取走保证金
     */
    function setWithdrawTimeLock(uint256 newWithdrawTimeLock) external onlyOwner {
        Operation.setWithdrawTimeLock(state, newWithdrawTimeLock);
    }

    /**
     * @notice 设置订单发送者（撮合引擎）
     * @param orderSender 订单发送者地址
     * @param isValid 是否有效
     * @dev 只有授权的订单发送者才能提交撮合交易
     *      通常是链下撮合引擎的地址
     */
    function setOrderSender(address orderSender, bool isValid) external onlyOwner {
        Operation.setOrderSender(state, orderSender, isValid);
    }

    /**
     * @notice 设置快速取款白名单
     * @param target 目标地址
     * @param isValid 是否在白名单中
     * @dev 白名单用户可以跳过取款时间锁，直接取款
     */
    function setFastWithdrawalWhitelist(address target, bool isValid) external onlyOwner {
        Operation.setFastWithdrawalWhitelist(state, target, isValid);
    }

    /**
     * @notice 设置取款白名单
     * @param target 目标地址
     * @param isValid 是否在白名单中
     */
    function setWithdrawlWhitelist(address target, bool isValid) external onlyOwner {
        Operation.setWithdrawalWhitelist(state, target, isValid);
    }

    /**
     * @notice 禁用/启用快速取款功能
     * @param disabled 是否禁用
     * @dev 在紧急情况下可以禁用快速取款
     */
    function disableFastWithdraw(bool disabled) external onlyOwner {
        Operation.disableFastWithdraw(state, disabled);
    }

    /**
     * @notice 设置次级资产地址
     * @param _secondaryAsset 次级资产地址
     * @dev 次级资产只能设置一次，必须与主资产小数位数相同
     *      次级资产可以用作额外的保证金类型
     */
    function setSecondaryAsset(address _secondaryAsset) external onlyOwner {
        Operation.setSecondaryAsset(state, _secondaryAsset);
    }
}
