/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

/**
 * @title IDecimalERC20 - 带精度查询的 ERC20 接口
 * @notice 定义获取 ERC20 代币小数位数的接口
 * @dev 用于验证主资产和次级资产的精度是否一致
 */
interface IDecimalERC20 {
    /**
     * @notice 获取代币的小数位数
     * @return 小数位数（如 USDC 返回 6，大多数代币返回 18）
     */
    function decimals() external returns (uint8);
}
