// +build manual

package coldwallet

import (
	"fmt"
	"testing"

	"github.com/thetatoken/theta/wallet/coldwallet"
	ks "github.com/thetatoken/theta/wallet/coldwallet/keystore"

	"github.com/stretchr/testify/assert"
)

func TestCreateColdWalletLedger(t *testing.T) {
	assert := assert.New(t)
	//TODO: change ledgerHub to trazorHub
	hub, err := newHub(LedgerScheme, 0x2c97, []uint16{0x0000 /* Ledger Blue */, 0x0001 /* Ledger Nano S */}, 0xf1d0, -1, ks.NewLedgerDriver)
	if err != nil {
		panic(fmt.Sprintf("Failed to create hub: %v", err))
	}
	cold_wallet, err := coldwallet.NewColdWallet()
	if err != nil {
		panic(fmt.Sprintf("Failed to create wallet: %v", err))
	}

	assert.NotNil(cold_wallet)
}
