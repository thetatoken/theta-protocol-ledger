package key

import (
	"github.com/spf13/cobra"
)

// KeyCmd represents the key command
var KeyCmd = &cobra.Command{
	Use:   "key",
	Short: "Manage keys",
	Long:  `Manage keys.`,
}

func init() {
	KeyCmd.AddCommand(newCmd)
	KeyCmd.AddCommand(listCmd)
	KeyCmd.AddCommand(deleteCmd)
	KeyCmd.AddCommand(passwordCmd)
}
