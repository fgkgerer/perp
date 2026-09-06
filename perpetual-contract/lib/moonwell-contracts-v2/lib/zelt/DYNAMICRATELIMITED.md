# DynamicRateLimitLibrary Documentation

## Overview

The `DynamicRateLimitLibrary` is a Solidity library that provides rate limit enforcement functionality to smart contracts. It utilizes the struct `DynamicRateLimit` to set and manage rate limits. It also introduces a feature that allows dynamic adjustment of buffer cap and rate limit per second proportional to the total value locked (TVL) in the contract.

This library is particularly useful for protocols that want to limit how quickly certain actions can occur, such as minting of tokens or request rates, based on the proportion of TVL involved in each action. Anytime the buffer is depleted or replenished, the bufferCap and rateLimitPerSecond are adjusted proportionally to the new TVL. This allows a graceful wind down or a slow increase in TVL for a protocol without sacrificing safety.

## Struct: DynamicRateLimit

A `DynamicRateLimit` struct holds the key parameters for rate limiting:

- `rateLimitPerSecond`: The rate limit applied per second.
- `bufferCap`: The maximum buffer that can be utilized at once.
- `lastBufferUsedTime`: The last timestamp when the buffer was used.
- `bufferStored`: The buffer available at the time of `lastBufferUsedTime`.

## functions

### buffer(RateLimit storage limit) public view returns (uint256)

Calculates the amount of the buffer remaining. It automatically replenishes at `rateLimitPerSecond` per second, up to `bufferCap`.

### depleteBuffer(DynamicRateLimit storage limit, uint256 amount, uint256 prevTvlAmount)

This method enforces the rate limit. It decreases the buffer by the "amount" specified. If the remaining buffer is less than the amount, it will revert. This method also adjusts the buffer cap and rate limit per second proportionally based on the new TVL.

### replenishBuffer(DynamicRateLimit storage limit, uint256 amount, uint256 prevTvlAmount)

Increases the buffer by the "amount" specified if under the `bufferCap`. If the buffer is already at the cap, no changes are made. This method also increases the buffer cap and rate limit per second proportionally based on the new TVL.

### sync(DynamicRateLimit storage limit)

Synchronizes the buffer to the current time. It should be called before any action that updates the buffer cap or rate limit per second.

### setRateLimitPerSecond(DynamicRateLimit storage limit, uint128 newRateLimitPerSecond)

Sets the rate limit per second. Updates the bufferStored and lastBufferUsedTime to account for all rate limits accrued before setting the new limit.

### setBufferCap(DynamicRateLimit storage limit, uint128 newBufferCap)

Sets the buffer cap. Before setting the new cap, it ensures to sync the buffer to account for all rate limits accrued.

## Events

- `BufferUsed`: Emitted when the buffer gets depleted.
- `BufferReplenished`: Emitted when the buffer gets replenished.
- `BufferCapUpdate`: Emitted when the buffer cap is updated.
- `RateLimitPerSecondUpdate`: Emitted when the rate limit per second is updated.

## Usage

To use the `DynamicRateLimitLibrary`, you should include it in your contract and call its methods providing a `DynamicRateLimit` struct instance. Adjust the `rateLimitPerSecond` and `bufferCap` values to fit the rate limiting requirements of your protocol. Please note, you need to manage and pass the previous total value locked (TVL) amounts to `depleteBuffer()` and `replenishBuffer()` for dynamic adjustments.
