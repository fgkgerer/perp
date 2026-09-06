package config

import "fmt"

// Sepolia 部署地址（与根目录 README.md 一致）。
const (
	SepoliaBTCPerp = "0x11Aae1f92Ff10bfbb205971e060CF6d9D917723b"
	SepoliaETHPerp = "0x98456DCbcEfea550293727A7E2DfD45De92740c0"
)

// ValidateSepoliaMarkets 校验 Markets 配置是否与 README Sepolia 地址一致。
func ValidateSepoliaMarkets(chainID int64, markets []MarketConfig) error {
	if chainID != 11155111 {
		return nil
	}
	expected := map[string]string{
		"BTC-PERP": SepoliaBTCPerp,
		"ETH-PERP": SepoliaETHPerp,
	}
	seen := make(map[string]bool, len(expected))
	for _, m := range markets {
		exp, ok := expected[m.Name]
		if !ok {
			continue
		}
		seen[m.Name] = true
		if !sameAddr(m.Address, exp) {
			return fmt.Errorf("Markets.%s 地址 %s 与 README Sepolia %s 不一致", m.Name, m.Address, exp)
		}
	}
	for name := range expected {
		if !seen[name] {
			return fmt.Errorf("Markets 缺少 %s（期望 %s）", name, expected[name])
		}
	}
	return nil
}

func sameAddr(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
