// SPDX-License-Identifier: MIT

pragma solidity ^0.8.19;

import "forge-std/Script.sol";

/**
 * @title Utils - 部署脚本工具库
 * @notice 提供部署脚本中常用的工具函数
 * 
 * 功能：
 * 1. 日志输出：格式化输出部署参数
 * 2. 类型转换：地址转字符串、字节转字符串
 */
library Utils {
    /**
     * @notice 日志输出函数输入参数
     * @param inputs 输入参数字符串数组
     * @dev 将所有参数拼接后输出到控制台
     *      用于调试和记录部署参数
     */
    function logInputs(string[] memory inputs) public view {
        string memory concatenatedInputs = "";
        for (uint256 i = 0; i < inputs.length; i++) {
            concatenatedInputs = string(abi.encodePacked(concatenatedInputs, inputs[i], " "));
        }
        console.log(concatenatedInputs);
    }

    /**
     * @notice 将地址转换为字符串
     * @param addr 要转换的地址
     * @return 地址的十六进制字符串（带 0x 前缀）
     * @dev 用于日志输出和字符串拼接
     * 
     * 示例：
     * addressToString(0x123...abc) => "0x123...abc"
     */
    function addressToString(address addr) internal pure returns (string memory) {
        bytes32 value = bytes32(uint256(uint160(addr)));
        bytes memory alphabet = "0123456789abcdef";

        bytes memory str = new bytes(42);
        str[0] = '0';
        str[1] = 'x';
        for (uint256 i = 0; i < 20; i++) {
            str[2 + i * 2] = alphabet[uint8(value[i + 12] >> 4)];
            str[3 + i * 2] = alphabet[uint8(value[i + 12] & 0x0f)];
        }
        return string(str);
    }

    /**
     * @notice 将字节数组转换为字符串（不带 0x 前缀）
     * @param data 要转换的字节数组
     * @return 字节的十六进制字符串
     * @dev 用于处理 bytes32 等原始字节数据
     * 
     * 示例：
     * bytesToStringWithout0x(0x1234) => "1234"
     */
    function bytesToStringWithout0x(bytes memory data) internal pure returns (string memory) {
        bytes memory alphabet = "0123456789abcdef";

        bytes memory str = new bytes(data.length * 2);
        for (uint256 i = 0; i < data.length; i++) {
            str[i * 2] = alphabet[uint8(data[i] >> 4)];
            str[i * 2 + 1] = alphabet[uint8(data[i] & 0x0f)];
        }
        return string(str);
    }
}
