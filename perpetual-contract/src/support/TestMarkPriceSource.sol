/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

/**
 * @title TestMarkPriceSource - 测试用价格源
 * @notice 仅用于测试环境的可手动设置价格的预言机
 * 
 * 特性：
 * - 任何人都可以设置价格（无权限限制）
 * - 实现 IPriceSource 接口
 * 
 * @dev 仅用于测试，不要在生产环境使用！
 */
contract TestMarkPriceSource {
    /// @notice 当前标记价格
    uint256 public price;

    /**
     * @notice 设置标记价格
     * @param _price 新价格
     */
    function setMarkPrice(uint256 _price) external {
        price = _price;
    }

    /**
     * @notice 获取标记价格
     * @return 当前标记价格
     */
    function getMarkPrice() external view returns (uint256) {
        return price;
    }

    /**
     * @notice 获取资产价格
     * @return 当前资产价格
     */
    function getAssetPrice() external view returns (uint256) {
        return price;
    }
}

