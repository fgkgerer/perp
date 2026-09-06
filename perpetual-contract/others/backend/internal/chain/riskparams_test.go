package chain

import (
	"context"
	"os"
	"testing"

	"metanode/internal/config"

	"github.com/ethereum/go-ethereum/common"
)

func TestGetRiskParamsSepolia(t *testing.T) {
	rpc := os.Getenv("SEPOLIA_RPC_URL")
	if rpc == "" {
		rpc = "https://ethereum-sepolia.publicnode.com"
	}
	dealer := os.Getenv("DEALER_ADDRESS")
	if dealer == "" {
		dealer = "0x62e738C8e807c5D8224044207ff7623F9e080Cd7"
	}
	perp := os.Getenv("BTC_PERP")
	if perp == "" {
		perp = "0x11Aae1f92Ff10bfbb205971e060CF6d9D917723b"
	}

	c, err := NewClient(config.EthereumConfig{
		RpcUrl:        rpc,
		ChainId:       11155111,
		DealerAddress: dealer,
	}, nil, config.ChainlinkConfig{})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	rp, err := c.GetRiskParams(context.Background(), common.HexToAddress(perp))
	if err != nil {
		t.Fatal(err)
	}
	if !rp.IsRegistered {
		t.Fatal("perp not registered")
	}
	if rp.InitialMarginRatio == nil || rp.InitialMarginRatio.Sign() == 0 {
		t.Fatal("empty initialMarginRatio")
	}
	t.Logf("imr=%s name=%s", rp.InitialMarginRatio, rp.Name)
}
