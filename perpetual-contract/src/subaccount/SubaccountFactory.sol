/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

import "@openzeppelin/contracts/proxy/Clones.sol";
import "./Subaccount.sol";

pragma solidity ^0.8.19;

/**
 * @title SubaccountFactory - 子账户工厂合约
 * @notice 用于创建和管理子账户的工厂合约
 * 
 * 设计模式：
 * 使用 EIP-1167 最小代理（Clone）模式部署子账户
 * - 优点：大幅降低部署 gas 成本（约 10 倍）
 * - 原理：所有 Clone 共享同一份逻辑代码，只存储状态
 * 
 * 功能：
 * 1. 创建新子账户
 * 2. 查询用户的所有子账户
 * 3. 按索引查询特定子账户
 * 
 * @dev 参考 https://eips.ethereum.org/EIPS/eip-1167
 */
contract SubaccountFactory {
    // ==================== 存储 ====================

    /// @notice 子账户模板合约地址（Clone 的实现合约）
    address immutable template;

    /// @notice 用户子账户注册表：用户地址 => 子账户地址数组
    /// @dev 子账户只能添加，不能删除
    mapping(address => address[]) subaccountRegistry;

    // ==================== 事件 ====================

    /**
     * @notice 新子账户创建事件
     * @param master 主账户地址（创建者）
     * @param subaccountIndex 子账户索引
     * @param subaccountAddress 子账户合约地址
     */
    event NewSubaccount(address indexed master, uint256 subaccountIndex, address subaccountAddress);

    // ==================== 构造函数 ====================

    /**
     * @notice 构造函数，部署模板合约
     * @dev 创建一个 Subaccount 实例作为模板
     *      模板的 owner 设为 factory 本身（防止被使用）
     */
    constructor() {
        template = address(new Subaccount());
        // 初始化模板，防止被直接使用
        Subaccount(template).init(address(this));
    }

    // ==================== 函数 ====================

    /**
     * @notice 创建新子账户
     * @return subaccount 新创建的子账户地址
     * @dev 使用 EIP-1167 Clone 模式部署
     *      子账户的 owner 设为 msg.sender
     * 
     * Gas 成本说明：
     * - 直接部署 Subaccount: ~200,000 gas
     * - Clone 部署: ~45,000 gas
     */
    function newSubaccount() external returns (address subaccount) {
        // 克隆模板合约
        subaccount = Clones.clone(template);
        // 初始化子账户，设置调用者为 owner
        Subaccount(subaccount).init(msg.sender);
        // 注册到用户的子账户列表
        subaccountRegistry[msg.sender].push(subaccount);
        emit NewSubaccount(msg.sender, subaccountRegistry[msg.sender].length - 1, subaccount);
    }

    /**
     * @notice 获取用户的所有子账户
     * @param master 用户地址
     * @return 子账户地址数组
     */
    function getSubaccounts(address master) external view returns (address[] memory) {
        return subaccountRegistry[master];
    }

    /**
     * @notice 按索引获取特定子账户
     * @param master 用户地址
     * @param index 子账户索引
     * @return 子账户地址
     */
    function getSubaccount(address master, uint256 index) external view returns (address) {
        return subaccountRegistry[master][index];
    }
}
