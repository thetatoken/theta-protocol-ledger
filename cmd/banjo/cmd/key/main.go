package key

import (
	"github.com/spf13/cobra"
)

// KeyNewCmd represents the keyNew command
var KeyCmd = &cobra.Command{
	Use:   "key",
	Short: "Manage keys",
	Long:  `Manage keys.`,
}

func init() {
	KeyCmd.AddCommand(newCmd)
	KeyCmd.AddCommand(listCmd)
	KeyCmd.AddCommand(deleteCmd)
}
