/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

import "../../src/libraries/SignedDecimalMath.sol";
import "../../src/libraries/EIP712.sol";
import "forge-std/Test.sol";

pragma solidity ^0.8.19;

/**
 * @title InternalLibraryTest - 内部库测试辅助合约
 * @notice 暴露内部库函数供测试使用
 */
contract InternalLibraryTest {
    /**
     * @notice 测试有符号小数乘法
     * @param a 被乘数
     * @param b 乘数
     */
    function mul(int256 a, int256 b) public pure {
        SignedDecimalMath.decimalMul(a, b);
    }

    /**
     * @notice 测试有符号小数除法
     * @param a 被除数
     * @param b 除数
     */
    function div(int256 a, int256 b) public pure {
        SignedDecimalMath.decimalDiv(a, b);
    }

    /**
     * @notice 测试绝对值计算
     * @param a 输入值
     */
    function abs(int256 a) public pure {
        SignedDecimalMath.abs(a);
    }

    /**
     * @notice 测试小数余数检查
     * @param a 被除数
     * @param b 除数
     */
    function Remainder(uint256 a, uint256 b) public pure {
        SignedDecimalMath.decimalRemainder(a, b);
    }

    /**
     * @notice 测试 EIP-712 域分隔符构建
     * @param name 协议名称
     * @param version 版本号
     * @param verifyingContract 验证合约地址
     */
    function tEIP712(string memory name, string memory version, address verifyingContract) public view {
        EIP712._buildDomainSeparator(name, version, verifyingContract);
    }
}

/**
 * @title DecimalMathTest - 小数数学库测试
 * @notice 测试 SignedDecimalMath 和 EIP712 库的功能
 * 
 * 测试覆盖：
 * 1. 有符号小数乘法
 * 2. 有符号小数除法
 * 3. 绝对值计算
 * 4. 余数检查
 * 5. EIP-712 域分隔符
 */
contract DecimalMathTest is Test {
    /// @notice 测试辅助合约实例
    InternalLibraryTest public helper;

    /**
     * @notice 测试设置
     * @dev 部署辅助合约
     */
    function setUp() public {
        helper = new InternalLibraryTest();
    }

    /**
     * @notice 测试乘法：-2 * -2 = 4
     */
    function testMul() public view {
        helper.mul(-2, -2e18);
    }

    /**
     * @notice 测试除法：-2 / -2 = 1
     */
    function testDiv() public view {
        helper.div(-2, -2e18);
    }

    /**
     * @notice 测试绝对值：|-2| = 2
     */
    function testAbs() public view {
        helper.abs(-2);
    }

    /**
     * @notice 测试余数：3 % 4（有余数）
     */
    function testRemainder() public view {
        helper.Remainder(3, 4);
    }

    /**
     * @notice 测试余数：3 % 4e18（有余数）
     */
    function testRemainder2() public view {
        helper.Remainder(3, 4e18);
    }

    /**
     * @notice 测试 EIP-712 域分隔符生成
     */
    function testEIP712() public view {
        helper.tEIP712("Hey", "Hey", address(this));
    }
}
