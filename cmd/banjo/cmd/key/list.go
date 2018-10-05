package key

import (
	"fmt"

	"github.com/spf13/cobra"
)

// listCmd represents the new command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all keys",
	Long:  `List all keys.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("key list called")
	},
}
