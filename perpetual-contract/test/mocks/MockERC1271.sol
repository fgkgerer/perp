/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

/**
 * @title MockERC1271 - 模拟 ERC-1271 合约签名验证
 * @notice 用于测试的智能合约钱包签名验证模拟
 * 
 * ERC-1271 是智能合约签名验证标准
 * 允许智能合约作为签名者验证签名有效性
 * 
 * 本模拟合约始终返回有效签名（0x1626ba7e）
 */
contract MockERC1271 {
    // 排除在覆盖率报告之外
    function test() public { }

    /**
     * @notice 验证签名
     * @return ERC-1271 魔术值，表示签名有效
     * @dev 0x1626ba7e = bytes4(keccak256("isValidSignature(bytes32,bytes)"))
     */
    function isValidSignature(bytes32, bytes memory) external pure returns (bytes4) {
        return 0x1626ba7e;  // 有效签名的魔术值
    }
}
