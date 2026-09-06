/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

/**
 * @title IPriceSource - 价格源接口
 * @notice 定义预言机适配器需要实现的接口
 * 
 * 价格类型说明：
 * - markPrice: 标记价格，用于计算未实现盈亏和清算
 *   通常是指数价格 + 基差调整
 * - assetPrice: 资产价格，用于计算抵押品价值
 * 
 * 在永续合约系统中，主要使用 getMarkPrice
 */
interface IPriceSource {
    /**
     * @notice 获取标记价格
     * @return price 标记价格（1e18 精度）
     * @dev 如果数据不可用会 revert
     *      标记价格用于：
     *      1. 计算未实现盈亏
     *      2. 计算保证金率
     *      3. 确定清算价格
     */
    function getMarkPrice() external view returns (uint256 price);

    /**
     * @notice 获取资产价格
     * @return 资产价格（1e18 精度）
     * @dev 如果数据不可用会 revert
     *      资产价格用于计算抵押品价值
     */
    function getAssetPrice() external view returns (uint256);
}
