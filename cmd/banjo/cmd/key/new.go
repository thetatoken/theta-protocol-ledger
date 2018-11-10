package key

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thetatoken/ukulele/cmd/banjo/cmd/utils"
	"github.com/thetatoken/ukulele/wallet"
)

// newCmd generates a new key
var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Generates a new private key",
	Long:  `Generates a new private key.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfgPath := cmd.Flag("config").Value.String()
		wallet, err := wallet.OpenDefaultWallet(cfgPath)
		if err != nil {
			utils.Error("Failed to open wallet: %v\n", err)
		}

		prompt := fmt.Sprintf("Please enter password: ")
		password, err := utils.GetPassword(prompt)
		if err != nil {
			utils.Error("Failed to get password: %v\n", err)
		}

		address, err := wallet.NewKey(password)
		if err != nil {
			utils.Error("Failed to generate new key: %v\n", err)
		}

		fmt.Printf("Successfully created key: %v\n", address.Hex())
	},
}
