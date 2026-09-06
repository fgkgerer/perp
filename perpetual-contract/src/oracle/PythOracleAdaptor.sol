/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/math/SafeCast.sol";
import "../interfaces/internal/IPyth.sol";
import "../interfaces/internal/IChainlink.sol";

/**
 * @title PythOracleAdaptor - Pyth + Chainlink 混合预言机适配器
 * @notice 结合 Pyth 和 Chainlink 两个预言机源，提供更可靠的价格数据
 * 
 * 设计理念：
 * - Chainlink 作为主要价格源（更稳定、更广泛使用）
 * - Pyth 作为补充（更高频、更低延迟）
 * - 当两个价格源偏差过大时，使用 Pyth 价格（可能更及时）
 * 
 * 价格选择逻辑：
 * 1. 获取 Chainlink 价格作为基准
 * 2. 尝试获取 Pyth 价格
 * 3. 如果 Pyth 不可用或与 Chainlink 偏差在阈值内，使用 Chainlink
 * 4. 如果偏差超过阈值，使用 Pyth（假设市场剧烈波动时 Pyth 更快反应）
 * 
 * @dev 合约名称与 OracleAdaptor.sol 相同是因为文件名不同，实际部署时选择其一
 */
contract OracleAdaptor is Ownable {
    // ==================== 状态变量 ====================
    
    /// @notice 精度校正因子
    uint256 public immutable decimalsCorrection;
    
    /// @notice Chainlink 心跳间隔
    uint256 public immutable heartbeatInterval;
    
    /// @notice USDC 心跳间隔
    uint256 public immutable usdcHeartbeat;
    
    /// @notice USDC Chainlink 价格源
    address public immutable usdcSource;
    
    /// @notice 主资产 Chainlink 价格源
    address public immutable chainlink;
    
    /// @notice Pyth 价格 ID（用于标识资产）
    bytes32 public immutable priceId;
    
    /// @notice 当前价格缓存
    uint256 public price;
    
    /// @notice 价格偏离阈值
    uint256 public priceThreshold;
    
    /// @notice Pyth 合约实例
    IPyth public pyth;

    // ==================== 事件 ====================
    
    /// @notice 阈值更新事件
    event UpdateThreshold(uint256 oldThreshold, uint256 newThreshold);

    // ==================== 构造函数 ====================
    
    /**
     * @notice 初始化混合预言机适配器
     * @param _chainlink Chainlink 价格源地址
     * @param _pythContract Pyth 合约地址
     * @param _decimalsCorrection 精度校正指数
     * @param _heartbeatInterval 主资产心跳间隔
     * @param _usdcHeartbeat USDC 心跳间隔
     * @param _usdcSource USDC Chainlink 价格源
     * @param _priceThreshold 价格偏离阈值
     * @param _priceId Pyth 价格 ID
     */
    constructor(
        address _chainlink,
        address _pythContract,
        uint256 _decimalsCorrection,
        uint256 _heartbeatInterval,
        uint256 _usdcHeartbeat,
        address _usdcSource,
        uint256 _priceThreshold,
        bytes32 _priceId
    ) {
        chainlink = _chainlink;
        pyth = IPyth(_pythContract);
        decimalsCorrection = 10 ** _decimalsCorrection;
        heartbeatInterval = _heartbeatInterval;
        usdcHeartbeat = _usdcHeartbeat;
        usdcSource = _usdcSource;
        priceThreshold = _priceThreshold;
        priceId = _priceId;
    }

    // ==================== 管理函数 ====================

    /**
     * @notice 更新价格偏离阈值
     * @param newPriceThreshold 新阈值
     */
    function updateThreshold(uint256 newPriceThreshold) external onlyOwner {
        priceThreshold = newPriceThreshold;
        emit UpdateThreshold(priceThreshold, newPriceThreshold);
    }

    // ==================== 价格查询函数 ====================

    /**
     * @notice 从 Chainlink 获取价格
     * @return 资产/USDC 价格（未调整精度）
     */
    function getChainLinkPrice() public view returns (uint256) {
        int256 rawPrice;
        uint256 updatedAt;
        (, rawPrice,, updatedAt,) = IChainlink(chainlink).latestRoundData();
        (, int256 usdcPrice,, uint256 usdcUpdatedAt,) = IChainlink(usdcSource).latestRoundData();
        require(block.timestamp - updatedAt <= heartbeatInterval, "ORACLE_HEARTBEAT_FAILED");
        require(block.timestamp - usdcUpdatedAt <= usdcHeartbeat, "USDC_ORACLE_HEARTBEAT_FAILED");
        uint256 tokenPrice = (SafeCast.toUint256(rawPrice) * 1e8) / SafeCast.toUint256(usdcPrice);
        return tokenPrice;
    }

    /**
     * @notice 获取价格（内部逻辑）
     * @return 选择的价格（未调整精度）
     * @dev 价格选择逻辑：
     *      1. 获取 Chainlink 价格
     *      2. 尝试获取 Pyth 价格
     *      3. 如果 Pyth 成功且偏差在阈值内，使用 Chainlink
     *      4. 如果偏差超过阈值，使用 Pyth
     *      5. 如果 Pyth 失败，使用 Chainlink
     */
    function getPrice() internal view returns (uint256) {
        uint256 chainLinkPrice = getChainLinkPrice();
        try pyth.getPrice(priceId) returns (PythStructs.Price memory pythPriceStruct) {
            uint256 pythPrice = SafeCast.toUint256(pythPriceStruct.price);
            uint256 diff = pythPrice >= chainLinkPrice ? pythPrice - chainLinkPrice : chainLinkPrice - pythPrice;
            if ((diff * 1e18) / chainLinkPrice <= priceThreshold) {
                // 偏差在阈值内，使用更稳定的 Chainlink 价格
                return chainLinkPrice;
            } else {
                // 偏差超过阈值，可能市场剧烈波动，使用更及时的 Pyth 价格
                return pythPrice;
            }
        } catch {
            // Pyth 不可用，回退到 Chainlink
            return chainLinkPrice;
        }
    }

    /**
     * @notice 获取标记价格
     * @return 标记价格（1e18 精度）
     */
    function getMarkPrice() external view returns (uint256) {
        return (getPrice() * 1e18) / decimalsCorrection;
    }

    /**
     * @notice 获取资产价格
     * @return 资产价格（1e18 精度）
     */
    function getAssetPrice() external view returns (uint256) {
        return (getPrice() * 1e18) / decimalsCorrection;
    }
}
