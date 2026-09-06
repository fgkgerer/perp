/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/math/SafeCast.sol";
import "../interfaces/internal/IChainlink.sol";

/**
 * @title OracleAdaptor - Chainlink 预言机适配器
 * @notice 为永续合约系统提供价格数据，支持 Chainlink 和自定义价格源
 * 
 * 功能特性：
 * 1. 从 Chainlink 获取实时价格
 * 2. 支持切换到自定义价格（带价格偏离保护）
 * 3. 使用 USDC 价格标准化报价
 * 4. 心跳检查确保价格新鲜度
 * 
 * 价格计算流程：
 * 1. 获取资产/USD 价格
 * 2. 获取 USDC/USD 价格
 * 3. 计算资产/USDC 价格 = (资产/USD) / (USDC/USD)
 * 4. 调整精度到 1e18
 * 
 * 安全机制：
 * - 心跳检查：价格更新时间不能超过设定的心跳间隔
 * - 价格偏离保护：自定义价格与 Chainlink 价格偏差不能过大
 */
contract OracleAdaptor is Ownable {
    // ==================== 状态变量 ====================
    
    /// @notice 精度校正因子，用于将价格转换为 1e18 精度
    uint256 public immutable decimalsCorrection;
    
    /// @notice 主资产价格的心跳间隔（秒），超过则认为价格过期
    uint256 public immutable heartbeatInterval;
    
    /// @notice USDC 价格的心跳间隔（秒）
    uint256 public immutable usdcHeartbeat;
    
    /// @notice USDC/USD 价格源地址（Chainlink）
    address public immutable usdcSource;
    
    /// @notice 主资产/USD 价格源地址（Chainlink）
    address public immutable chainlink;
    
    /// @notice 当前轮次 ID（用于自定义价格更新追踪）
    uint256 public roundId;
    
    /// @notice 自定义价格（MetaNode 自己设置的价格）
    uint256 public price;
    
    /// @notice 价格偏离阈值（1e18 基数），超过此阈值则拒绝自定义价格
    uint256 public priceThreshold;
    
    /// @notice 是否使用自定义价格源
    bool public isSelfOracle;

    // ==================== 事件 ====================
    
    /// @notice 价格更新事件（与 Chainlink 格式对齐，便于监控）
    event AnswerUpdated(int256 indexed current, uint256 indexed roundId, uint256 updatedAt);
    
    /// @notice 价格偏离阈值更新事件
    event UpdateThreshold(uint256 oldThreshold, uint256 newThreshold);

    // ==================== 构造函数 ====================
    
    /**
     * @notice 初始化预言机适配器
     * @param _chainlink 主资产的 Chainlink 价格源地址
     * @param _decimalsCorrection 精度校正指数（如 8 表示 Chainlink 返回 8 位小数）
     * @param _heartbeatInterval 主资产心跳间隔（秒）
     * @param _usdcHeartbeat USDC 心跳间隔（秒）
     * @param _usdcSource USDC 的 Chainlink 价格源地址
     * @param _priceThreshold 价格偏离阈值（1e18 基数）
     */
    constructor(
        address _chainlink,
        uint256 _decimalsCorrection,
        uint256 _heartbeatInterval,
        uint256 _usdcHeartbeat,
        address _usdcSource,
        uint256 _priceThreshold
    ) {
        chainlink = _chainlink;
        decimalsCorrection = 10 ** _decimalsCorrection;
        heartbeatInterval = _heartbeatInterval;
        usdcHeartbeat = _usdcHeartbeat;
        usdcSource = _usdcSource;
        priceThreshold = _priceThreshold;
    }

    // ==================== 管理函数 ====================

    /**
     * @notice 设置自定义标记价格
     * @param newPrice 新价格（1e18 精度）
     * @dev 仅管理员可调用
     *      需要先开启 isSelfOracle 才会生效
     */
    function setMarkPrice(uint256 newPrice) external onlyOwner {
        price = newPrice;
        emit AnswerUpdated(SafeCast.toInt256(price), roundId, block.timestamp);
        roundId += 1;
    }

    /**
     * @notice 开启自定义价格源
     * @dev 开启后将使用 setMarkPrice 设置的价格
     *      但仍会检查与 Chainlink 价格的偏离度
     */
    function turnOnMetaNodeOracle() external onlyOwner {
        isSelfOracle = true;
    }

    /**
     * @notice 关闭自定义价格源，使用 Chainlink 价格
     */
    function turnOffMetaNodeOracle() external onlyOwner {
        isSelfOracle = false;
    }

    /**
     * @notice 更新价格偏离阈值
     * @param newPriceThreshold 新阈值（1e18 基数，如 0.03e18 表示 3%）
     */
    function updateThreshold(uint256 newPriceThreshold) external onlyOwner {
        priceThreshold = newPriceThreshold;
        emit UpdateThreshold(priceThreshold, newPriceThreshold);
    }

    // ==================== 价格查询函数 ====================

    /**
     * @notice 从 Chainlink 获取价格
     * @return 资产/USDC 价格（1e18 精度）
     * @dev 计算过程：
     *      1. 获取资产/USD 价格（rawPrice）
     *      2. 获取 USDC/USD 价格（usdcPrice）
     *      3. tokenPrice = rawPrice / usdcPrice（以 USDC 计价）
     *      4. 转换为 1e18 精度
     */
    function getChainLinkPrice() public view returns (uint256) {
        int256 rawPrice;
        uint256 updatedAt;
        // 获取主资产价格
        (, rawPrice,, updatedAt,) = IChainlink(chainlink).latestRoundData();
        // 获取 USDC 价格
        (, int256 usdcPrice,, uint256 usdcUpdatedAt,) = IChainlink(usdcSource).latestRoundData();
        // 心跳检查
        require(block.timestamp - updatedAt <= heartbeatInterval, "ORACLE_HEARTBEAT_FAILED");
        require(block.timestamp - usdcUpdatedAt <= usdcHeartbeat, "USDC_ORACLE_HEARTBEAT_FAILED");
        // 计算以 USDC 计价的价格
        uint256 tokenPrice = (SafeCast.toUint256(rawPrice) * 1e8) / SafeCast.toUint256(usdcPrice);
        // 转换为 1e18 精度
        return (tokenPrice * 1e18) / decimalsCorrection;
    }

    /**
     * @notice 获取价格（内部函数）
     * @return 最终使用的价格（1e18 精度）
     * @dev 如果开启了自定义价格源：
     *      1. 获取 Chainlink 价格作为基准
     *      2. 计算自定义价格与基准的偏差
     *      3. 如果偏差在阈值内，使用自定义价格
     *      4. 否则 revert
     *      如果未开启自定义价格源，直接返回 Chainlink 价格
     */
    function getPrice() internal view returns (uint256) {
        uint256 chainLinkPrice = getChainLinkPrice();
        if (isSelfOracle) {
            uint256 MetaNodePrice = price;
            // 计算价格偏差
            uint256 diff = MetaNodePrice >= chainLinkPrice ? MetaNodePrice - chainLinkPrice : chainLinkPrice - MetaNodePrice;
            // 检查偏差是否在阈值内
            require((diff * 1e18) / chainLinkPrice <= priceThreshold, "deviation is too big");
            return price;
        } else {
            return chainLinkPrice;
        }
    }

    /**
     * @notice 获取标记价格（对外接口）
     * @return 标记价格（1e18 精度）
     */
    function getMarkPrice() external view returns (uint256) {
        return getPrice();
    }

    /**
     * @notice 获取资产价格（对外接口）
     * @return 资产价格（1e18 精度）
     */
    function getAssetPrice() external view returns (uint256) {
        return getPrice();
    }
}
