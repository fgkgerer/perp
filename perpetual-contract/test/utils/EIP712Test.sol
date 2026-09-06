// SPDX-License-Identifier: BUSL-1.1

pragma solidity >=0.8.0;

import "../../src/libraries/Types.sol";

/**
 * @title EIP712Test - 测试用 EIP-712 签名工具
 * @notice 提供用于测试的 EIP-712 签名相关函数
 * 
 * 功能：
 * 1. 计算订单结构体哈希
 * 2. 构建域分隔符
 * 3. 计算类型数据哈希
 * 
 * @dev 这些函数与 src/libraries/EIP712.sol 中的实现相同
 *      放在测试目录是为了避免测试依赖内部实现
 */
library EIP712Test {
    // 排除在覆盖率报告之外
    function test() public { }

    /**
     * @notice 计算订单的结构体哈希
     * @param order 订单结构体
     * @return structHash 结构体哈希
     * @dev 使用汇编优化 gas 消耗
     */
    function _structHash(Types.Order memory order) internal pure returns (bytes32 structHash) {
        bytes32 orderTypeHash = Types.ORDER_TYPEHASH;
        assembly {
            let start := sub(order, 32)
            let tmp := mload(start)
            // 192 = (1 + 5) * 32
            // [0...32)   bytes: EIP712_ORDER_TYPE
            // [32...192) bytes: order
            mstore(start, orderTypeHash)
            structHash := keccak256(start, 192)
            mstore(start, tmp)
        }
    }

    /**
     * @notice 构建 EIP-712 域分隔符
     * @param name 协议名称
     * @param version 版本号
     * @param verifyingContract 验证合约地址
     * @return 域分隔符哈希
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
        bytes32 typeHash =
            keccak256("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)");
        return keccak256(abi.encode(typeHash, hashedName, hashedVersion, block.chainid, verifyingContract));
    }

    /**
     * @notice 计算 EIP-712 类型数据哈希
     * @param domainSeparator 域分隔符
     * @param structHash 结构体哈希
     * @return 最终的消息哈希（用于签名）
     */
    function _hashTypedDataV4(bytes32 domainSeparator, bytes32 structHash) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked("\x19\x01", domainSeparator, structHash));
    }
}
