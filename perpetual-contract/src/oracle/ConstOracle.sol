/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

/**
 * @title ConstOracle - 固定价格预言机
 * @notice 提供恒定不变的价格，主要用于测试和特殊场景
 * 
 * 使用场景：
 * 1. 测试环境：提供稳定的测试价格
 * 2. 稳定币市场：某些稳定币可以使用固定价格 1e18
 * 3. 模拟环境：在模拟测试中使用固定价格
 * 
 * @dev 部署后价格不可更改，因为是 immutable
 */
contract ConstOracle {
    /// @notice 固定价格（1e18 精度）
    uint256 public immutable price;

    /**
     * @notice 构造函数，设置固定价格
     * @param _price 价格值（1e18 精度）
     *        例如：1e18 表示 1 美元，30000e18 表示 30000 美元
     */
    constructor(uint256 _price) {
        price = _price;
    }

    /**
     * @notice 获取标记价格
     * @return 固定的价格值
     */
    function getMarkPrice() external view returns (uint256) {
        return price;
    }
}
