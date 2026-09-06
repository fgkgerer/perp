/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

/**
 * @title MockUSDCPrice - 模拟 USDC 价格预言机
 * @notice 用于测试的 USDC/USD 价格预言机模拟
 * 
 * 返回固定价格：$1.00（10^8 = 1 * 10^8）
 * 用于将其他资产价格标准化为 USDC 计价
 */
contract MockUSDCPrice {
    // 排除在覆盖率报告之外
    function test() public { }

    /**
     * @notice 获取最新价格数据
     * @return roundId 轮次 ID
     * @return answer 价格（$1.00，8位小数）
     * @return startedAt 开始时间
     * @return updatedAt 更新时间
     * @return answeredInRound 回答轮次
     * @dev USDC 锚定美元，价格固定为 1
     */
    function latestRoundData()
        external
        pure
        returns (uint80 roundId, int256 answer, uint256 startedAt, uint256 updatedAt, uint80 answeredInRound)
    {
        return (1, 100_000_000, 1, 1, 1);  // $1.00 * 10^8
    }
}
