package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/consensus"
	"github.com/thetatoken/ukulele/node"
	"github.com/thetatoken/ukulele/p2p/messenger"
	p2ptypes "github.com/thetatoken/ukulele/p2p/types"
	"github.com/thetatoken/ukulele/store"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start Theta node.",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		start()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}

func start() {
	network := newMessenger([]string{}, "127.0.0.1:4666")
	validators := consensus.NewTestValidatorSet([]string{"v1", "v2", "v3", "v4", network.ID()})

	store := store.NewMemKVStore()
	chainID := "testchain"
	root := &blockchain.Block{}
	root.ChainID = chainID
	root.Epoch = 0
	root.Hash = blockchain.ParseHex("a0")

	params := &node.Params{
		Store:      store,
		ChainID:    chainID,
		Root:       root,
		Validators: validators,
		Network:    network,
	}
	n := node.NewNode(params)
	n.Start(context.Background())

	n.Wait()
}

func newMessenger(seedPeerNetAddressStrs []string, localNetworkAddress string) *messenger.Messenger {
	peerPubKey := p2ptypes.GetTestRandPubKey()
	peerNodeInfo := p2ptypes.CreateNodeInfo(peerPubKey)
	addrbookPath := "./.addrbooks/addrbook_" + localNetworkAddress + ".json"
	routabilityRestrict := false
	selfNetAddressStr := "104.105.23.92:8888" // not important for the test
	networkProtocol := "tcp"
	skipUPNP := true
	messenger, err := messenger.CreateMessenger(peerNodeInfo, addrbookPath, routabilityRestrict, selfNetAddressStr,
		seedPeerNetAddressStrs, networkProtocol, localNetworkAddress, skipUPNP)
	if err != nil {
		panic(fmt.Sprintf("Failed to create PeerDiscoveryManager instance: %v", err))
	}
	return messenger
}
