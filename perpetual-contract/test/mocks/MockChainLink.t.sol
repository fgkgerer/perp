/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

/**
 * @title MockChainLink - 模拟 Chainlink 预言机
 * @notice 用于测试的 Chainlink 价格预言机模拟
 * 
 * 返回固定价格：$1000（10^11 = 1000 * 10^8）
 * Chainlink 通常使用 8 位小数
 */
contract MockChainLink {
    // 排除在覆盖率报告之外
    function test() public { }

    /**
     * @notice 获取最新价格数据
     * @return roundId 轮次 ID
     * @return answer 价格（$1000，8位小数）
     * @return startedAt 开始时间
     * @return updatedAt 更新时间
     * @return answeredInRound 回答轮次
     */
    function latestRoundData()
        external
        pure
        returns (uint80 roundId, int256 answer, uint256 startedAt, uint256 updatedAt, uint80 answeredInRound)
    {
        return (1, 100_000_000_000, 1, 1, 1);  // $1000 * 10^8
    }
}
