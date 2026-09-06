# RateLimitedLibrary

`RateLimitedLibrary` is a Solidity library that allows a contract to limit how fast certain actions (e.g., token minting) can be performed, effectively implementing a rate limit. It's a component of an Ethereum smart contract that helps in managing and enforcing rate limits.

## Structs

### RateLimit

This struct contains all information about the rate limit, including:

- `rateLimitPerSecond`: The rate limit per second for the contract.
- `bufferCap`: The maximum buffer size that can be utilized at once.
- `lastBufferUsedTime`: The last time the buffer was used by the contract.
- `bufferStored`: The buffer at the timestamp of `lastBufferUsedTime`.

## Functions

### buffer(RateLimit storage limit) public view returns (uint256)

This function returns the buffer size at the current time. It calculates this based on the elapsed time since the last buffer use and the `rateLimitPerSecond`, up to the `bufferCap`.

### depleteBuffer(RateLimit storage limit, uint256 amount) internal

This function decreases the buffer size by the specified `amount`. If the buffer size is less than or equal to the `amount`, the function reverts. An event `BufferUsed` is emitted every time this function is called.

### replenishBuffer(RateLimit storage limit, uint256 amount) internal

This function increases the buffer size by the specified `amount` up to the `bufferCap`. If the buffer is already at the cap, the function does nothing. An event `BufferReplenished` is emitted every time this function is called.

### sync(RateLimit storage limit) internal

This function syncs the buffer to the current block timestamp. It's advised to call this function before any action that updates buffer cap or rate limit per second.

### setRateLimitPerSecond(RateLimit storage limit, uint128 newRateLimitPerSecond) internal

This function sets a new rate limit per second. It first syncs the buffer before applying the new limit. An event `RateLimitPerSecondUpdate` is emitted every time this function is called.

### setBufferCap(RateLimit storage limit, uint128 newBufferCap) internal

This function sets a new buffer cap. It first syncs the buffer before applying the new cap. An event `BufferCapUpdate` is emitted every time this function is called.

## Events

- `BufferUsed`: Emitted when the buffer is depleted.
- `BufferReplenished`: Emitted when the buffer is replenished.
- `BufferCapUpdate`: Emitted when the buffer cap is updated.
- `RateLimitPerSecondUpdate`: Emitted when the rate limit per second is updated.

## Usage

To use this library, import it into your contract and use its functions to apply rate limits on specific actions. You can create a `RateLimit` struct instance in your contract to manage the rate limits. Remember to set the buffer cap and rate limit per second before performing any rate-limited operations.

Remember to handle rate-limited operations carefully as exceeding the rate limit or buffer cap will cause transactions to fail. 

This library helps you ensure that your contract doesn't perform certain actions too quickly, providing a simple way to enforce rate limits and manage related data in Solidity.
