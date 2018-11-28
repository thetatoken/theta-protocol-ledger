package key

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thetatoken/ukulele/cmd/banjo/cmd/utils"
	"github.com/thetatoken/ukulele/wallet"
)

// listCmd lists all the stored keys
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all keys",
	Long:  `List all keys.`,
	Example: "banjo key list",
	Run: func(cmd *cobra.Command, args []string) {
		cfgPath := cmd.Flag("config").Value.String()
		wallet, err := wallet.OpenDefaultWallet(cfgPath)
		if err != nil {
			utils.Error("Failed to open wallet: %v\n", err)
		}

		keyAddresses, err := wallet.List()
		if err != nil {
			utils.Error("Failed to list keys: %v\n", err)
		}

		for _, keyAddress := range keyAddresses {
			fmt.Printf("%s\n", keyAddress.Hex())
		}
	},
}
