/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

/**
 * @title PythStructs - Pyth 预言机数据结构
 * @notice 定义 Pyth 预言机使用的数据结构
 * 
 * Pyth 是一个高频率更新的预言机网络
 * 特别适合需要低延迟价格数据的 DeFi 应用
 */
contract PythStructs {
    /**
     * @notice 带置信区间的价格结构
     * @dev 置信区间大致对应正态分布的标准误差
     *      价格和置信度使用定点数表示：x * (10^expo)
     * 
     * 安全使用价格的最佳实践请参考：
     * https://docs.pyth.network/documentation/pythnet-price-feeds/best-practices
     */
    struct Price {
        /// @notice 价格值
        int64 price;
        /// @notice 价格的置信区间
        uint64 conf;
        /// @notice 价格指数（用于确定精度）
        int32 expo;
        /// @notice 价格发布的 Unix 时间戳
        uint256 publishTime;
    }

    /**
     * @notice 价格馈送结构
     * @dev 包含从 Pyth 发布者聚合的当前价格
     */
    struct PriceFeed {
        /// @notice 价格 ID（唯一标识一个价格馈送）
        bytes32 id;
        /// @notice 最新可用价格
        Price price;
        /// @notice 最新可用的指数加权移动平均价格
        Price emaPrice;
    }
}

/**
 * @title IPyth - Pyth 预言机接口
 * @notice 定义与 Pyth 价格预言机交互的接口
 * 
 * 特点：
 * - 高频率更新（亚秒级）
 * - 拉取模式（用户主动更新价格）
 * - 支持多种资产
 */
interface IPyth {
    /**
     * @notice 获取价格和置信区间
     * @param id Pyth 价格馈送 ID
     * @return price 价格结构体
     * @dev 如果价格在 getValidTimePeriod() 秒内未更新则 revert
     *      请仔细阅读 PythStructs.Price 文档以安全使用价格
     */
    function getPrice(bytes32 id) external view returns (PythStructs.Price memory price);

    /**
     * @notice 更新价格馈送
     * @param updateData 价格更新数据数组
     * @dev 调用者需要支付 Wei 格式的费用
     *      费用可通过 getUpdateFee 计算
     *      只有比当前存储价格更新的价格才会被更新
     *      即使更新不是最新的，调用也会成功
     * @dev 如果转账费用不足或 updateData 无效则 revert
     */
    function updatePriceFeeds(bytes[] calldata updateData) external payable;

    /**
     * @notice 条件更新价格馈送
     * @param updateData 价格更新数据数组
     * @param priceIds 价格 ID 数组
     * @param publishTimes 发布时间数组，与 priceIds 一一对应
     * @dev 如果不需要更新则快速拒绝以节省 gas
     *      当链上 publishTime 比给定的 publishTime 旧时才更新
     * 
     *      priceIds 和 publishTimes 是相同大小的数组
     *      对应调用者已知的每个 priceId 的 publishTime
     *      如果所有价格馈送都已更新且有更新或相等的发布时间
     *      则拒绝交易以节省 gas
     */
    function updatePriceFeedsIfNecessary(
        bytes[] calldata updateData,
        bytes32[] calldata priceIds,
        uint64[] calldata publishTimes
    )
        external
        payable;

    /**
     * @notice 获取更新价格所需的费用
     * @param updateData 价格更新数据数组
     * @return feeAmount 所需费用（Wei）
     */
    function getUpdateFee(bytes[] calldata updateData) external view returns (uint256 feeAmount);
}
