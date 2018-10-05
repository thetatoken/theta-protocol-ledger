package key

import (
	"fmt"

	"github.com/spf13/cobra"
)

// deleteCmd represents the new command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a key",
	Long:  `Delete a key`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("key delete called")
	},
}
