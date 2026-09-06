/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/security/ReentrancyGuard.sol";
import "./libraries/EIP712.sol";
import "./libraries/Errors.sol";
import "./libraries/Types.sol";

/**
 * @title MetaNodeStorage - 永续合约系统存储层
 * @notice 包含 MetaNodeDealer 的所有状态变量和访问控制修饰符
 * 
 * 状态说明：
 * - state: 包含所有核心状态的结构体（定义在 Types.sol）
 * - domainSeparator: EIP-712 签名域分隔符，用于订单签名验证
 * 
 * 访问控制：
 * - onlyFundingRateKeeper: 仅资金费率更新者可调用
 * - onlyRegisteredPerp: 仅已注册的永续合约可调用
 */
abstract contract MetaNodeStorage is Ownable, ReentrancyGuard {
    /// @notice 系统核心状态，包含用户余额、市场参数、订单记录等
    Types.State public state;

    /// @notice EIP-712 域分隔符，用于订单签名验证，确保签名不能跨链/跨合约重放
    bytes32 public immutable domainSeparator;

    /**
     * @notice 构造函数，初始化 EIP-712 域分隔符
     * @dev 域分隔符包含协议名称、版本、合约地址，确保签名唯一性
     */
    constructor() Ownable() {
        domainSeparator = EIP712._buildDomainSeparator("MetaNode", "1", address(this));
    }

    /**
     * @notice 资金费率更新者权限检查
     * @dev 只有指定的 fundingRateKeeper 地址才能更新资金费率
     */
    modifier onlyFundingRateKeeper() {
        require(msg.sender == state.fundingRateKeeper, Errors.INVALID_FUNDING_RATE_KEEPER);
        _;
    }

    /**
     * @notice 已注册永续合约权限检查
     * @dev 只有已在系统注册的 Perpetual 合约才能调用某些函数
     *      用于确保只有合法的交易市场才能进行结算操作
     */
    modifier onlyRegisteredPerp() {
        require(state.perpRiskParams[msg.sender].isRegistered, Errors.PERP_NOT_REGISTERED);
        _;
    }
}
