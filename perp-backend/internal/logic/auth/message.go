package auth

import "fmt"

// BuildWalletLoginMessage 与 verify 侧校验用的明文一致（EIP-191 personal_sign）。
func BuildWalletLoginMessage(walletChecksum string, nonce string) string {
	return fmt.Sprintf("Sign in to MetaNode.\n\nWallet: %s\nNonce: %s", walletChecksum, nonce)
}
