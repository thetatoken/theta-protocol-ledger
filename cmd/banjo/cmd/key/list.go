package key

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thetatoken/ukulele/wallet"
)

// listCmd lists all the stored keys
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all keys",
	Long:  `List all keys.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfgPath := cmd.Flag("config").Value.String()
		wallet, err := wallet.OpenDefaultWallet(cfgPath)
		if err != nil {
			fmt.Printf("Failed to open wallet: %v\n", err)
			return
		}

		keyAddresses, err := wallet.List()
		if err != nil {
			fmt.Printf("Failed to list keys: %v\n", err)
			return
		}

		for _, keyAddress := range keyAddresses {
			fmt.Printf("%s\n", keyAddress.Hex())
		}
	},
}
