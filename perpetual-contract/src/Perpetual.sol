/*
    Copyright 2022 MetaNode Protocol
     SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/math/SafeCast.sol";
import "./interfaces/IDealer.sol";
import "./interfaces/IPerpetual.sol";
import "./libraries/SignedDecimalMath.sol";

/**
 * @title Perpetual - 永续合约市场
 * @notice 单个永续合约市场的核心资产负债表合约
 * 
 * 核心概念：
 * - paper: 仓位数量，正数为多头，负数为空头
 * - credit: 资金量，与 paper 配合计算盈亏
 * - reducedCredit: 约减后的 credit，用于优化存储
 * - fundingRate: 资金费率（累计值）
 * 
 * 关键公式：
 * credit = (paper * fundingRate) + reducedCredit
 * 
 * 举例：
 * - 以 $30,000 做多 1 BTC: paper = 1e18, credit = -30000e6
 * - 以 $30,000 做空 1 BTC: paper = -1e18, credit = 30000e6
 * 
 * 存储优化：
 * 使用 int128 存储 paper 和 reducedCredit，可以在一个 slot 中存储
 * 计算时转换为 int256 以保证精度
 */
contract Perpetual is Ownable, IPerpetual {
    using SignedDecimalMath for int256;

    // ==================== 存储 ====================

    /**
     * @dev 余额结构体，使用 int128 节省 gas
     *      int128 最大值约 1.7e38，对于大多数交易足够
     *      paper 通常是 1e18 基数的小数
     */
    struct balance {
        int128 paper;          // 仓位数量
        int128 reducedCredit;  // 约减后的资金量
    }

    /// @notice 交易者地址 => 余额
    mapping(address => balance) balanceMap;
    
    /// @notice 当前资金费率（累计值）
    int256 fundingRate;

    // ==================== 事件 ====================

    /// @notice 余额变化事件
    /// @param trader 交易者地址
    /// @param paperChange paper 变化量
    /// @param creditChange credit 变化量
    event BalanceChange(address indexed trader, int256 paperChange, int256 creditChange);

    /// @notice 资金费率更新事件
    /// @param oldFundingRate 旧资金费率
    /// @param newFundingRate 新资金费率
    event UpdateFundingRate(int256 oldFundingRate, int256 newFundingRate);

    // ==================== 构造函数 ====================

    /**
     * @notice 构造函数
     * @param _owner 所有者地址（通常是 MetaNodeDealer）
     * @dev Perpetual 的 owner 是 Dealer，这样 Dealer 可以调用特权函数
     */
    constructor(address _owner) Ownable() {
        transferOwnership(_owner);
    }

    // ==================== 余额相关 ====================

    /*
        存储优化说明：
        我们存储 reducedCredit 而不是直接存储 credit
        这样在资金费率更新后，credit 值会自动更新，无需额外的存储写入
        
        公式：credit = (paper * fundingRate) + reducedCredit
        
        资金费率说明：
        这里的 fundingRate 与 CEX 的含义略有不同
        fundingRate 是累计值，其绝对值没有意义，只有变化量才重要
        
        例如：如果 fundingRate 在某次更新中增加了 5
        - 每持有 1 paper 多头，你将获得 5 credit
        - 每持有 1 paper 空头，你将支付 5 credit
    */

    /**
     * @notice 查询交易者在此市场的余额
     * @param trader 交易者地址
     * @return paper 仓位数量（正=多头，负=空头）
     * @return credit 资金量（包含资金费率调整）
     */
    function balanceOf(address trader) external view returns (int256 paper, int256 credit) {
        paper = int256(balanceMap[trader].paper);
        // 使用公式计算实际 credit: credit = paper * fundingRate + reducedCredit
        credit = paper.decimalMul(fundingRate) + int256(balanceMap[trader].reducedCredit);
    }

    /**
     * @notice 更新资金费率（仅 owner/Dealer 可调用）
     * @param newFundingRate 新的资金费率
     * @dev 资金费率的变化会自动影响所有持仓者的 credit
     *      多头在资金费率上升时获益，空头在资金费率下降时获益
     */
    function updateFundingRate(int256 newFundingRate) external onlyOwner {
        int256 oldFundingRate = fundingRate;
        fundingRate = newFundingRate;
        emit UpdateFundingRate(oldFundingRate, newFundingRate);
    }

    /**
     * @notice 获取当前资金费率
     * @return 资金费率（累计值）
     */
    function getFundingRate() external view returns (int256) {
        return fundingRate;
    }

    // ==================== 交易 ====================

    /**
     * @notice 执行交易
     * @param tradeData 编码的交易数据
     * @dev 交易流程：
     *      1. 调用 Dealer.approveTrade 验证订单并计算结果
     *      2. 结算每个交易者的余额变化
     *      3. 检查所有交易者的安全性
     */
    function trade(bytes calldata tradeData) external {
        // 调用 Dealer 验证交易并获取结算结果
        (address[] memory traderList, int256[] memory paperChangeList, int256[] memory creditChangeList) =
            IDealer(owner()).approveTrade(msg.sender, tradeData);

        // 结算每个交易者的余额
        for (uint256 i = 0; i < traderList.length;) {
            _settle(traderList[i], paperChangeList[i], creditChangeList[i]);
            unchecked {
                ++i;
            }
        }

        // 确保所有交易者交易后都是安全的
        require(IDealer(owner()).isAllSafe(traderList), "TRADER_NOT_SAFE");
    }

    // ==================== 清算 ====================

    /**
     * @notice 清算不安全的仓位
     * @param liquidator 清算者地址
     * @param liquidatedTrader 被清算者地址
     * @param requestPaper 请求清算的仓位数量
     * @param expectCredit 期望的 credit 变化（用于价格保护）
     * @return liqtorPaperChange 清算者的 paper 变化
     * @return liqtorCreditChange 清算者的 credit 变化
     * 
     * @dev 清算流程：
     *      1. 向 Dealer 请求清算，获取结算金额
     *      2. 验证价格保护（防止清算价格过差）
     *      3. 结算双方余额
     *      4. 检查清算者安全性
     *      5. 如果被清算者仓位归零，处理可能的坏账
     * 
     * 清算机制说明：
     * - 当交易者的保证金率低于维持保证金率时，可被清算
     * - 清算者以一定折扣接手被清算者的仓位
     * - 价格保护机制防止清算价格偏离太多
     */
    function liquidate(
        address liquidator,
        address liquidatedTrader,
        int256 requestPaper,
        int256 expectCredit
    )
        external
        returns (int256 liqtorPaperChange, int256 liqtorCreditChange)
    {
        // liqed => 被清算者，面临清算风险
        // liqtor => 清算者，接手仓位的人
        int256 liqedPaperChange;
        int256 liqedCreditChange;
        (liqtorPaperChange, liqtorCreditChange, liqedPaperChange, liqedCreditChange) =
            IDealer(owner()).requestLiquidation(msg.sender, liquidator, liquidatedTrader, requestPaper);

        // 价格保护检查
        // 期望价格 = expectCredit / requestPaper * -1
        // 执行价格 = liqtorCreditChange / liqtorPaperChange * -1
        if (liqtorPaperChange < 0) {
            // 开空仓，执行价格 >= 期望价格
            require(
                liqtorCreditChange * requestPaper <= expectCredit * liqtorPaperChange, 
                "LIQUIDATION_PRICE_PROTECTION"
            );
        } else {
            // 开多仓，执行价格 <= 期望价格
            require(
                liqtorCreditChange * requestPaper >= expectCredit * liqtorPaperChange, 
                "LIQUIDATION_PRICE_PROTECTION"
            );
        }

        // 结算双方余额
        _settle(liquidatedTrader, liqedPaperChange, liqedCreditChange);
        _settle(liquidator, liqtorPaperChange, liqtorCreditChange);
        
        // 确保清算者是安全的
        require(IDealer(owner()).isSafe(liquidator), "LIQUIDATOR_NOT_SAFE");
        
        // 如果被清算者仓位归零，检查并处理坏账
        if (balanceMap[liquidatedTrader].paper == 0) {
            IDealer(owner()).handleBadDebt(liquidatedTrader);
        }
    }

    // ==================== 结算 ====================

    /*
        结算公式推导：
        credit = (paper * fundingRate) + reducedCredit
        
        因此：
        reducedCredit = credit - (paper * fundingRate)
        
        更新余额时：
        1. 先计算新的 credit
        2. 再计算并存储 reducedCredit
    */

    /**
     * @notice 内部结算函数
     * @param trader 交易者地址
     * @param paperChange paper 变化量
     * @param creditChange credit 变化量
     * @dev 结算流程：
     *      1. 计算新的 credit（包含 fundingRate 调整）
     *      2. 计算新的 paper
     *      3. 反推并存储 reducedCredit
     *      4. 触发事件
     *      5. 如果是新仓位，通知 Dealer
     *      6. 如果仓位归零，实现盈亏
     */
    function _settle(address trader, int256 paperChange, int256 creditChange) internal {
        bool isNewPosition = balanceMap[trader].paper == 0;
        int256 rate = fundingRate; // 缓存以节省 gas
        
        // 计算新的 credit
        int256 credit =
            int256(balanceMap[trader].paper).decimalMul(rate) + int256(balanceMap[trader].reducedCredit) + creditChange;
        
        // 计算新的 paper
        int128 newPaper = balanceMap[trader].paper + SafeCast.toInt128(paperChange);
        
        // 反推 reducedCredit: reducedCredit = credit - paper * fundingRate
        int128 newReducedCredit = SafeCast.toInt128(credit - int256(newPaper).decimalMul(rate));
        
        // 更新存储
        balanceMap[trader].paper = newPaper;
        balanceMap[trader].reducedCredit = newReducedCredit;
        
        emit BalanceChange(trader, paperChange, creditChange);
        
        // 如果是新开仓，通知 Dealer 记录
        if (isNewPosition) {
            IDealer(owner()).openPosition(trader);
        }
        
        // 如果仓位归零，实现盈亏
        if (newPaper == 0) {
            // 将剩余的 reducedCredit 作为实现盈亏转入账户
            IDealer(owner()).realizePnl(trader, balanceMap[trader].reducedCredit);
            balanceMap[trader].reducedCredit = 0;
        }
    }
}
