/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";

/**
 * @title Subaccount - 子账户合约
 * @notice 帮助用户管理风险和仓位的智能合约账户
 * 
 * 核心功能：
 * 1. 隔离仓位：通过子账户开仓，与主账户风险隔离
 * 2. 委托交易：可以授权操作员代为交易，但无资金操作权限
 * 3. 灵活执行：可以调用任意合约函数
 * 
 * 使用场景：
 * - 机构用户：多个交易员使用独立子账户，风险隔离
 * - 量化交易：授权机器人操作，但限制资金权限
 * - 仓位管理：不同策略使用不同子账户
 * 
 * 安全设计：
 * - 所有权不可转让（防止权限滥用）
 * - 只能初始化一次（防止重入攻击）
 * - 仅所有者可执行操作
 * 
 * @dev 使用可初始化模式支持 Clone，降低部署 gas 成本
 */
contract Subaccount {
    // ==================== 存储 ====================

    /**
     * @notice 子账户所有者
     * @dev 与标准 Ownable 不同，所有权不可转让
     *      设计为可初始化以支持 Clone 模式
     */
    address public owner;
    
    /// @notice 是否已初始化
    bool public initialized;

    // ==================== 修饰符 ====================

    /**
     * @notice 仅所有者可调用
     */
    modifier onlyOwner() {
        require(owner == msg.sender, "Ownable: caller is not the owner");
        _;
    }

    // ==================== 事件 ====================
    
    /**
     * @notice 执行交易事件
     * @param owner 子账户所有者
     * @param subaccount 子账户地址
     * @param to 目标合约地址
     * @param data 调用数据
     * @param value 发送的 ETH 数量
     */
    event ExecuteTransaction(address indexed owner, address subaccount, address to, bytes data, uint256 value);

    // ==================== 函数 ====================

    /**
     * @notice 初始化子账户
     * @param _owner 所有者地址
     * @dev 只能调用一次
     *      由 SubaccountFactory 在 clone 后调用
     */
    function init(address _owner) external {
        require(!initialized, "ALREADY INITIALIZED");
        initialized = true;
        owner = _owner;
    }

    /**
     * @notice 执行任意合约调用
     * @param to 目标合约地址
     * @param data 调用数据（编码后的函数调用）
     * @param value 随调用发送的 ETH 数量
     * @return 调用返回的数据
     * 
     * @dev 仅所有者可调用
     *      支持发送 ETH（如果需要）
     *      失败时会回滚并转发错误信息
     * 
     * 使用示例：
     * - 调用 Dealer.deposit: execute(dealer, abi.encodeWithSelector(IDealer.deposit.selector, amount, 0, address(this)), 0)
     * - 设置操作员: execute(dealer, abi.encodeWithSelector(IDealer.setOperator.selector, operator, true), 0)
     */
    function execute(
        address to,
        bytes calldata data,
        uint256 value
    )
        external
        payable
        onlyOwner
        returns (bytes memory)
    {
        require(to != address(0), "execute address is empty");
        require(msg.value == value, "TRANSFER_PAYMENT_NOT_MATCH");
        (bool success, bytes memory returnData) = to.call{ value: value }(data);
        if (!success) {
            // 转发错误信息
            assembly {
                let ptr := mload(0x40)
                let size := returndatasize()
                returndatacopy(ptr, 0, size)
                revert(ptr, size)
            }
        }
        emit ExecuteTransaction(owner, address(this), to, data, value);
        return returnData;
    }
}
