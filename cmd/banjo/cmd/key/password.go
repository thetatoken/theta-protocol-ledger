package key

import (
	"fmt"

	"github.com/spf13/cobra"
)

// passwordCmd changes the password for a key
var passwordCmd = &cobra.Command{
	Use:   "password",
	Short: "Change the password for a key",
	Long:  `Change the password for a key.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("key password called")
	},
}
