// SPDX-License-Identifier: MIT

pragma solidity ^0.8.19;

/**
 * @title WETH9 - Wrapped ETH 模拟合约
 * @notice 用于测试的 WETH 代币实现
 * 
 * WETH (Wrapped ETH) 将原生 ETH 包装成 ERC-20 代币
 * 使 ETH 能够在需要 ERC-20 接口的 DeFi 协议中使用
 * 
 * 功能：
 * - deposit: 存入 ETH 获得 WETH
 * - withdraw: 销毁 WETH 取回 ETH
 * - 标准 ERC-20 转账功能
 */
contract WETH9 {
    // 排除在覆盖率报告之外
    function test() public { }

    /// @notice 代币名称
    string public name = "Wrapped Ether";
    /// @notice 代币符号
    string public symbol = "WETH";
    /// @notice 小数位数
    uint8 public decimals = 18;

    // ==================== 事件 ====================

    /// @notice 授权事件
    event Approval(address indexed src, address indexed guy, uint256 wad);
    /// @notice 转账事件
    event Transfer(address indexed src, address indexed dst, uint256 wad);
    /// @notice 存款事件（ETH -> WETH）
    event Deposit(address indexed dst, uint256 wad);
    /// @notice 取款事件（WETH -> ETH）
    event Withdrawal(address indexed src, uint256 wad);

    // ==================== 状态变量 ====================

    /// @notice 用户余额
    mapping(address => uint256) public balanceOf;
    /// @notice 授权额度
    mapping(address => mapping(address => uint256)) public allowance;

    // ==================== 接收 ETH ====================

    /**
     * @notice 接收 ETH 时自动存款
     */
    fallback() external payable {
        deposit();
    }

    receive() external payable {
        deposit();
    }

    // ==================== 核心功能 ====================

    /**
     * @notice 存入 ETH 获得 WETH
     * @dev 1:1 兑换
     */
    function deposit() public payable {
        balanceOf[msg.sender] += msg.value;
        emit Deposit(msg.sender, msg.value);
    }

    /**
     * @notice 销毁 WETH 取回 ETH
     * @param wad 取款数量
     * @dev 1:1 兑换
     */
    function withdraw(uint256 wad) public {
        require(balanceOf[msg.sender] >= wad);
        balanceOf[msg.sender] -= wad;
        payable(msg.sender).transfer(wad);
        emit Withdrawal(msg.sender, wad);
    }

    /**
     * @notice 获取总供应量
     * @return 合约持有的 ETH 数量
     */
    function totalSupply() public view returns (uint256) {
        return address(this).balance;
    }

    // ==================== ERC-20 标准功能 ====================

    /**
     * @notice 授权代币
     * @param guy 被授权者
     * @param wad 授权额度
     * @return 是否成功
     */
    function approve(address guy, uint256 wad) public returns (bool) {
        allowance[msg.sender][guy] = wad;
        emit Approval(msg.sender, guy, wad);
        return true;
    }

    /**
     * @notice 转账
     * @param dst 接收者
     * @param wad 转账数量
     * @return 是否成功
     */
    function transfer(address dst, uint256 wad) public returns (bool) {
        return transferFrom(msg.sender, dst, wad);
    }

    /**
     * @notice 授权转账
     * @param src 来源地址
     * @param dst 目标地址
     * @param wad 转账数量
     * @return 是否成功
     */
    function transferFrom(address src, address dst, uint256 wad) public returns (bool) {
        require(balanceOf[src] >= wad);
        // 检查授权（无限授权不扣减）
        if (src != msg.sender && allowance[src][msg.sender] != type(uint256).max) {
            require(allowance[src][msg.sender] >= wad);
            allowance[src][msg.sender] -= wad;
        }

        balanceOf[src] -= wad;
        balanceOf[dst] += wad;

        emit Transfer(src, dst, wad);

        return true;
    }

    /// @notice 测试辅助函数
    function testSuccess() public { }
}
