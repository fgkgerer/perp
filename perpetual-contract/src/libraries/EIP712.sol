/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

import "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import "./Types.sol";

/**
 * @title EIP712 - 结构化数据签名库
 * @notice 实现 EIP-712 签名标准，用于订单签名验证
 * 
 * EIP-712 简介：
 * - 以太坊签名标准，让用户签名时能看到结构化数据
 * - 包含域分隔符（防止跨链/跨合约重放）
 * - 包含类型哈希（确保数据结构一致）
 * 
 * 在本系统中的应用：
 * - 用户在链下签名订单
 * - 签名包含订单详情（市场、数量、价格、过期时间等）
 * - 链上验证签名有效性
 * 
 * @dev 更多信息参见 https://eips.ethereum.org/EIPS/eip-712
 */
library EIP712 {
    /**
     * @notice 构建 EIP-712 域分隔符
     * @param name 协议名称
     * @param version 协议版本
     * @param verifyingContract 验证合约地址
     * @return 域分隔符的哈希值
     * 
     * @dev 域分隔符包含：
     *      - name: 协议名称（"MetaNode"）
     *      - version: 版本号（"1"）
     *      - chainId: 链 ID（防止跨链重放）
     *      - verifyingContract: 合约地址（防止跨合约重放）
     * 
     * 域分隔符在合约部署时计算并存储为 immutable
     * 确保同一签名不能在其他合约或链上使用
     */
    function _buildDomainSeparator(
        string memory name,
        string memory version,
        address verifyingContract
    )
        internal
        view
        returns (bytes32)
    {
        bytes32 hashedName = keccak256(bytes(name));
        bytes32 hashedVersion = keccak256(bytes(version));
        // EIP-712 标准域类型哈希
        bytes32 typeHash =
            keccak256("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)");
        return keccak256(abi.encode(typeHash, hashedName, hashedVersion, block.chainid, verifyingContract));
    }

    /**
     * @notice 计算 EIP-712 类型数据哈希
     * @param domainSeparator 域分隔符
     * @param structHash 结构体哈希
     * @return 最终的签名消息哈希
     * 
     * @dev 格式："\x19\x01" || domainSeparator || structHash
     *      这是 EIP-712 规定的标准格式
     *      "\x19\x01" 是 EIP-712 的前缀，用于与其他签名类型区分
     */
    function _hashTypedDataV4(bytes32 domainSeparator, bytes32 structHash) internal pure returns (bytes32) {
        return ECDSA.toTypedDataHash(domainSeparator, structHash);
    }
}
