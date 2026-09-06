/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

/**
 * @title MockERC1271Failed - 模拟失败的 ERC-1271 签名验证
 * @notice 用于测试签名验证失败场景
 * 
 * 本模拟合约始终返回无效的魔术值
 * 用于测试签名验证失败的处理逻辑
 */
contract MockERC1271Failed {
    // 排除在覆盖率报告之外
    function test() public { }

    /**
     * @notice 验证签名（始终返回无效）
     * @return 错误的魔术值，表示签名无效
     * @dev 返回 0x1626ba72 而不是正确的 0x1626ba7e
     */
    function isValidSignature(bytes32, bytes memory) external pure returns (bytes4) {
        return 0x1626ba72;  // 无效的魔术值
    }
}
