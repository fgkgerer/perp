# Solidity Rate Limiting Libraries

## Overview

Solidity Rate Limiting Libraries, composed of the `DynamicRateLimitLibrary` and the `RateLimitedLibrary`, provide essential functionality for Ethereum smart contracts to enforce rate limits on certain operations. These libraries are particularly beneficial in controlling the pace of operations, such as token minting or request rates. This management of contract's operational speed is achieved through a structured approach involving a rate limit per second and a buffer cap.

## Libraries

### DynamicRateLimitLibrary

`DynamicRateLimitLibrary` introduces a unique feature that allows dynamic adjustment of the buffer cap and rate limit per second proportional to the total value locked (TVL) in the contract. This is especially useful for protocols that need a graceful wind down or a slow increase in TVL without sacrificing safety.

It provides several functions to enforce and manage the rate limits, such as `buffer()`, `depleteBuffer()`, `replenishBuffer()`, `sync()`, `setRateLimitPerSecond()`, and `setBufferCap()`. These functions work around a struct called `DynamicRateLimit`, which holds key parameters for rate limiting.

Every time the buffer is depleted or replenished, the `bufferCap` and `rateLimitPerSecond` are adjusted proportionally to the new TVL. As a result, the buffer cap and rate limit are dynamic and adapt to the contract's TVL.

### RateLimitedLibrary

`RateLimitedLibrary` provides a basic yet powerful means to apply rate limits on specific actions. This library helps to manage and enforce rate limits, ensuring your contract doesn't perform certain actions too quickly. It contains a `RateLimit` struct to manage the rate limits and several functions to enforce and manage these limits, similar to `DynamicRateLimitLibrary`.

This library offers a simple way to enforce rate limits and manage related data in Solidity. By using it, you can create a safety buffer for your operations and prevent transaction failures due to rate limit exceedances.

## Usage

To implement these libraries in your contract, import them and create instances of the respective struct (either `DynamicRateLimit` or `RateLimit`). Adjust the `rateLimitPerSecond` and `bufferCap` values to suit your protocol's needs. For `DynamicRateLimitLibrary`, you'll need to manage and pass the previous total value locked (TVL) amounts to `depleteBuffer()` and `replenishBuffer()` for dynamic adjustments.

Using this design, it is possible to create multiple rate limits in a single contract to enforce different rate limits on different actions.

Remember to handle rate-limited operations carefully as exceeding the rate limit or buffer cap will cause transactions to fail.

See [RateLimited.sol](./src/RateLimited.sol) for an example of how to use the RateLimited library.
