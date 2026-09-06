/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

import "../init/TradingInit.sol";
import "../utils/Checkers.sol";

/**
 * @title ScenarioDepositWithdrawTest - 存取款场景测试
 * @author MetaNode Team
 * @notice 演示保证金存取的完整流程
 * 
 * ============================================================
 *                      存取款机制说明
 * ============================================================
 * 
 * 存款：
 *   - 无需许可，直接调用 deposit()
 *   - 支持主资产（USDC）和次级资产（MUSD）
 *   - 可存入自己账户或他人账户
 * 
 * 取款：
 *   两种模式：
 *   1. 待定取款（Pending Withdraw）
 *      - 先 requestWithdraw()
 *      - 等待 withdrawTimeLock
 *      - 再 executeWithdraw()
 *   2. 快速取款（Fast Withdraw）
 *      - 需要白名单权限
 *      - 一步完成取款
 * 
 * ============================================================
 *                   前端开发参考
 * ============================================================
 * 
 * 存款流程：
 * 1. 调用 USDC.approve(dealer, amount)
 * 2. 调用 dealer.deposit(primaryAmount, secondaryAmount, to)
 * 
 * 取款流程：
 * 1. 调用 dealer.requestWithdraw(primaryAmount, secondaryAmount)
 * 2. 等待 withdrawTimeLock 秒
 * 3. 调用 dealer.executeWithdraw(user, to, isInternal)
 * 
 * 余额查询：
 * - dealer.getCreditOf(user) 返回完整资金状态
 */
contract ScenarioDepositWithdrawTest is Checkers {
    /// @notice 存款金额
    uint256 constant DEPOSIT_AMOUNT = 10_000e6;
    
    // ==================== 场景1: 基础存款 ====================
    
    /**
     * @notice 场景1：基础存款流程
     * @dev 演示如何存入 USDC 作为保证金
     */
    function testScenario1_BasicDeposit() public {
        address alice = traders[0];
        
        // 查询存款前余额
        uint256 usdcBefore = usdc.balanceOf(alice);
        (int256 creditBefore,,,,) = metaNodeDealer.getCreditOf(alice);
        
        // 执行存款
        vm.startPrank(alice);
        // 步骤1: 授权（前端需要先检查授权额度）
        usdc.approve(address(metaNodeDealer), DEPOSIT_AMOUNT);
        
        // 步骤2: 存款
        metaNodeDealer.deposit(DEPOSIT_AMOUNT, 0, alice);
        vm.stopPrank();
        
        // 验证存款后余额
        uint256 usdcAfter = usdc.balanceOf(alice);
        (int256 creditAfter,,,,) = metaNodeDealer.getCreditOf(alice);
        
        assertEq(usdcBefore - usdcAfter, DEPOSIT_AMOUNT, "USDC should decrease");
        assertEq(creditAfter - creditBefore, int256(DEPOSIT_AMOUNT), "Credit should increase");
    }
    
    // ==================== 场景2: 次级资产存款 ====================
    
    /**
     * @notice 场景2：存入次级资产（MUSD）
     * @dev 次级资产可以作为额外保证金
     */
    function testScenario2_SecondaryAssetDeposit() public {
        address alice = traders[0];
        
        vm.startPrank(alice);
        // 存入主资产和次级资产
        metaNodeDealer.deposit(5_000e6, 5_000e6, alice);
        vm.stopPrank();
        
        // 查询余额
        (int256 primaryCredit, uint256 secondaryCredit,,,) = metaNodeDealer.getCreditOf(alice);
        
        assertEq(primaryCredit, 5_000e6, "Primary credit should be 5000");
        assertEq(secondaryCredit, 5_000e6, "Secondary credit should be 5000");
    }
    
    // ==================== 场景3: 待定取款 ====================
    
    /**
     * @notice 场景3：待定取款流程
     * @dev 演示完整的两步取款流程
     */
    function testScenario3_PendingWithdraw() public {
        address alice = traders[0];
        
        // 先存款
        vm.startPrank(alice);
        metaNodeDealer.deposit(DEPOSIT_AMOUNT, 0, alice);
        
        // ========== 步骤1: 请求取款 ==========
        uint256 withdrawAmount = 5_000e6;
        metaNodeDealer.requestWithdraw(alice, withdrawAmount, 0);
        
        // 查询待取款状态
        (
            int256 primaryCredit,
            ,
            uint256 pendingPrimaryWithdraw,
            ,
            uint256 executionTimestamp
        ) = metaNodeDealer.getCreditOf(alice);
        
        assertEq(pendingPrimaryWithdraw, withdrawAmount, "Pending amount should match");
        
        // ========== 步骤2: 等待时间锁 ==========
        // 注意：如果 withdrawTimeLock 为 0，可能不会 revert
        // 这里跳过立即取款测试，直接等待后执行
        
        // 快进时间
        vm.warp(executionTimestamp + 1);
        
        // ========== 步骤3: 执行取款 ==========
        uint256 usdcBefore = usdc.balanceOf(alice);
        metaNodeDealer.executeWithdraw(alice, alice, false, "");
        uint256 usdcAfter = usdc.balanceOf(alice);
        
        vm.stopPrank();
        
        // 验证取款成功
        assertEq(usdcAfter - usdcBefore, withdrawAmount, "Should receive withdraw amount");
        
        // 验证待取款已清空
        (,, uint256 pendingAfter,,) = metaNodeDealer.getCreditOf(alice);
        assertEq(pendingAfter, 0, "Pending should be cleared");
    }
    
    // ==================== 场景4: 有仓位时取款限制 ====================
    
    /**
     * @notice 场景4：有仓位时的取款限制
     * @dev 不能取走维持保证金
     */
    function testScenario4_WithdrawWithPosition() public {
        address alice = traders[0];
        address bob = traders[1];
        
        // 存款
        vm.prank(alice);
        metaNodeDealer.deposit(DEPOSIT_AMOUNT, 0, alice);
        vm.prank(bob);
        metaNodeDealer.deposit(DEPOSIT_AMOUNT, 0, bob);
        
        // 开仓：Alice 做多 1 BTC
        trade(1e18, -30_000e6, -1e18, 30_000e6, 1e18, 1e18, address(perpList[0]));
        
        // 尝试取出大部分保证金
        vm.startPrank(alice);
        metaNodeDealer.requestWithdraw(alice, 9_000e6, 0);  // 尝试取 $9,000
        
        (,,,, uint256 execTime) = metaNodeDealer.getCreditOf(alice);
        vm.warp(execTime + 1);
        
        // 应该因为保证金不足而失败
        vm.expectRevert();  // 会因为保证金检查失败
        metaNodeDealer.executeWithdraw(alice, alice, false, "");
        vm.stopPrank();
    }
    
    // ==================== 场景5: 存款到他人账户 ====================
    
    /**
     * @notice 场景5：为他人存款
     * @dev 支持将保证金存入其他地址
     */
    function testScenario5_DepositForOther() public {
        address alice = traders[0];
        address bob = traders[1];
        
        (int256 bobCreditBefore,,,,) = metaNodeDealer.getCreditOf(bob);
        
        // Alice 为 Bob 存款
        vm.startPrank(alice);
        usdc.approve(address(metaNodeDealer), DEPOSIT_AMOUNT);
        metaNodeDealer.deposit(DEPOSIT_AMOUNT, 0, bob);  // to = bob
        vm.stopPrank();
        
        (int256 bobCreditAfter,,,,) = metaNodeDealer.getCreditOf(bob);
        
        assertEq(bobCreditAfter - bobCreditBefore, int256(DEPOSIT_AMOUNT));
    }
    
    // ==================== 场景6: 内部转账取款 ====================
    
    /**
     * @notice 场景6：内部转账（取款到其他交易者）
     * @dev isInternal = true 时，资金转移到另一个交易者账户
     */
    function testScenario6_InternalTransfer() public {
        address alice = traders[0];
        address bob = traders[1];
        
        // Alice 存款
        vm.startPrank(alice);
        metaNodeDealer.deposit(DEPOSIT_AMOUNT, 0, alice);
        
        // Alice 请求取款
        metaNodeDealer.requestWithdraw(alice, 5_000e6, 0);
        
        (,,,, uint256 execTime) = metaNodeDealer.getCreditOf(alice);
        vm.warp(execTime + 1);
        
        // 内部转账到 Bob
        (int256 bobBefore,,,,) = metaNodeDealer.getCreditOf(bob);
        metaNodeDealer.executeWithdraw(alice, bob, true, "");  // isInternal = true
        (int256 bobAfter,,,,) = metaNodeDealer.getCreditOf(bob);
        
        vm.stopPrank();
        
        assertEq(bobAfter - bobBefore, 5_000e6, "Bob should receive transfer");
    }
}
