package key

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thetatoken/ukulele/cmd/banjo/cmd/utils"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/wallet"
)

// deleteCmd represents the new command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a key",
	Long:  `Delete a key`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Printf("Usage: banjo key password <address>\n")
			return
		}
		address := common.HexToAddress(args[0])

		cfgPath := cmd.Flag("config").Value.String()
		wallet, err := wallet.OpenDefaultWallet(cfgPath)
		if err != nil {
			fmt.Printf("Failed to open wallet: %v\n", err)
			return
		}

		prompt := fmt.Sprintf("Please enter the password: ")
		password, err := utils.GetPassword(prompt)
		if err != nil {
			fmt.Printf("Failed to get password: %v\n", err)
			return
		}

		prompt = fmt.Sprintf("Are you sure to delete the key? Please enter the password again to proceed: ")
		password2, err := utils.GetPassword(prompt)
		if err != nil {
			fmt.Printf("Failed to get password: %v\n", err)
			return
		}

		if password != password2 {
			fmt.Printf("Passwords do not match, abort\n")
			return
		}

		err = wallet.Delete(address, password)
		if err != nil {
			fmt.Printf("Failed to delete key for address %v: %v\n", address.Hex(), err)
			return
		}

		fmt.Printf("Key for address %v has been deleted\n", address.Hex())
	},
}
