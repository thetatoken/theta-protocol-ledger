package key

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thetatoken/ukulele/cmd/banjo/cmd/utils"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/wallet"
)

// passwordCmd updates the password for the key corresponding to the given address
var passwordCmd = &cobra.Command{
	Use:   "password",
	Short: "Change the password for a key",
	Long:  `Change the password for a key.`,
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

		prompt := fmt.Sprintf("Please enter the current password: ")
		oldPassword, err := utils.GetPassword(prompt)
		if err != nil {
			fmt.Printf("Failed to get password: %v\n", err)
			return
		}

		prompt = fmt.Sprintf("Please enter a new password: ")
		newPassword, err := utils.GetPassword(prompt)
		if err != nil {
			fmt.Printf("Failed to get password: %v\n", err)
			return
		}

		err = wallet.UpdatePassword(address, oldPassword, newPassword)
		if err != nil {
			fmt.Printf("Failed to update password: %v\n", err)
			return
		}

		fmt.Printf("Password updated successfully\n")
	},
}
