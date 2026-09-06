/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

/**
 * @title TestERC20 - 测试用 ERC20 代币
 * @notice 仅用于测试环境的 ERC20 代币实现
 * 
 * 特性：
 * - 任何人都可以铸造代币（无权限限制）
 * - 可自定义名称、符号、小数位数
 * 
 * @dev 仅用于测试，不要在生产环境使用！
 */
contract TestERC20 is ERC20 {
    /// @notice 代币小数位数
    uint8 private _decimals;

    /**
     * @notice 构造函数
     * @param name_ 代币名称
     * @param symbol_ 代币符号
     * @param decimals_ 小数位数
     */
    constructor(string memory name_, string memory symbol_, uint8 decimals_) ERC20(name_, symbol_) {
        _decimals = decimals_;
    }

    /**
     * @notice 获取代币小数位数
     * @return 小数位数
     */
    function decimals() public view virtual override returns (uint8) {
        return _decimals;
    }

    /**
     * @notice 铸造代币
     * @param to 接收地址
     * @param amount 铸造数量
     * @dev 任何人都可以调用（仅用于测试）
     */
    function mint(address to, uint256 amount) external {
        _mint(to, amount);
    }

    /**
     * @notice 销毁代币
     * @param from 销毁来源地址
     * @param amount 销毁数量
     */
    function burn(address from, uint256 amount) external {
        _burn(from, amount);
    }
}

