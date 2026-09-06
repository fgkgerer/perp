/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

/**
 * @title IChainlink - Chainlink 预言机接口
 * @notice 定义与 Chainlink 价格预言机交互的接口
 * 
 * Chainlink 是最广泛使用的去中心化预言机网络
 * 提供可靠的链下数据到链上的传输
 * 
 * @dev 完整接口参见 Chainlink 官方文档
 *      https://docs.chain.link/data-feeds/price-feeds
 */
interface IChainlink {
    /**
     * @notice 获取最新一轮的价格数据
     * @return roundId 当前轮次 ID
     * @return answer 价格（需要根据 decimals 调整精度）
     * @return startedAt 本轮开始时间戳
     * @return updatedAt 最后更新时间戳
     * @return answeredInRound 回答所在的轮次
     * 
     * @dev 使用注意事项：
     *      1. 检查 updatedAt 确保数据新鲜度
     *      2. 检查 answer > 0 确保价格有效
     *      3. 考虑不同预言机有不同的 decimals
     */
    function latestRoundData()
        external
        view
        returns (uint80 roundId, int256 answer, uint256 startedAt, uint256 updatedAt, uint80 answeredInRound);
}
