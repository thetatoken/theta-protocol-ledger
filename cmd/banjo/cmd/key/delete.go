package key

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thetatoken/ukulele/cmd/banjo/cmd/utils"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/wallet"
)

// deleteCmd deletes the key corresponding to the given address
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a key",
	Long:  `Delete a key`,
	Example: "banjo delete 26d813157F7503a9057FB2DB6Eb2f83a35c4FdD7",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			utils.Error("Usage: banjo key <address>\n")
		}
		address := common.HexToAddress(args[0])

		cfgPath := cmd.Flag("config").Value.String()
		wallet, err := wallet.OpenDefaultWallet(cfgPath)
		if err != nil {
			utils.Error("Failed to open wallet: %v\n", err)
		}

		prompt := fmt.Sprintf("Please enter the password: ")
		password, err := utils.GetPassword(prompt)
		if err != nil {
			utils.Error("Failed to get password: %v\n", err)
		}

		// todo: option to abort deletion
		prompt = fmt.Sprintf("Are you sure to delete the key? Please enter the password again to proceed: ")
		password2, err := utils.GetPassword(prompt)
		if err != nil {
			utils.Error("Failed to get password: %v\n", err)
		}

		if password != password2 {
			utils.Error("Passwords do not match, abort\n")
		}

		err = wallet.Delete(address, password)
		if err != nil {
			utils.Error("Failed to delete key for address %v: %v\n", address.Hex(), err)
		}

		fmt.Printf("Key for address %v has been deleted\n", address.Hex())
	},
}
