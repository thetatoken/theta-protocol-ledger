package key

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newCmd represents the new command
var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Generates a new private key",
	Long:  `Generates a new private key.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("key new called")
	},
}
