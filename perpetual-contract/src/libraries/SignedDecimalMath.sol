/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

/**
 * @title SignedDecimalMath - 有符号小数数学库
 * @notice 提供 18 位精度的有符号/无符号小数运算
 * 
 * 设计说明：
 * - 使用 1e18 作为基数（ONE = 10^18）
 * - 支持有符号整数（int256）和无符号整数（uint256）
 * - 所有运算向下取整（round down）
 * 
 * 使用示例：
 * - 1.5 表示为 1.5e18 = 1500000000000000000
 * - 10% 表示为 0.1e18 = 100000000000000000
 * 
 * @dev 注意：乘法可能溢出，除法不检查除零
 *      调用方需确保参数在安全范围内
 */
library SignedDecimalMath {
    /// @notice 有符号的基数 (1e18)
    int256 constant SignedONE = 10 ** 18;
    /// @notice 无符号的基数 (1e18)
    uint256 constant ONE = 1e18;

    /**
     * @notice 有符号小数乘法
     * @param a 被乘数
     * @param b 乘数
     * @return a * b / 1e18
     * @dev 示例：1.5e18 * 2e18 / 1e18 = 3e18
     */
    function decimalMul(int256 a, int256 b) internal pure returns (int256) {
        return (a * b) / SignedONE;
    }

    /**
     * @notice 有符号小数除法
     * @param a 被除数
     * @param b 除数
     * @return a * 1e18 / b
     * @dev 示例：3e18 * 1e18 / 2e18 = 1.5e18
     */
    function decimalDiv(int256 a, int256 b) internal pure returns (int256) {
        return (a * SignedONE) / b;
    }

    /**
     * @notice 计算绝对值
     * @param a 输入值
     * @return 绝对值（总是非负）
     * @dev 注意：int256 最小值的绝对值会溢出
     *      但在实际业务场景中不会出现这种极端情况
     */
    function abs(int256 a) internal pure returns (uint256) {
        return a < 0 ? uint256(a * -1) : uint256(a);
    }

    /**
     * @notice 无符号小数乘法
     * @param a 被乘数
     * @param b 乘数
     * @return a * b / 1e18
     */
    function decimalMul(uint256 a, uint256 b) internal pure returns (uint256) {
        return (a * b) / ONE;
    }

    /**
     * @notice 无符号小数除法
     * @param a 被除数
     * @param b 除数
     * @return a * 1e18 / b
     */
    function decimalDiv(uint256 a, uint256 b) internal pure returns (uint256) {
        return (a * ONE) / b;
    }

    /**
     * @notice 检查小数除法是否有余数
     * @param a 被除数
     * @param b 除数
     * @return 是否整除（无余数）
     * @dev 用于检查某些精度要求严格的场景
     */
    function decimalRemainder(uint256 a, uint256 b) internal pure returns (bool) {
        if ((a * ONE) % b == 0) {
            return true;
        } else {
            return false;
        }
    }
}
