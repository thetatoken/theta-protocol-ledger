package account

import (
	"github.com/spf13/cobra"
)

// AccountCmd represents the account command
var AccountCmd = &cobra.Command{
	Use:   "account",
	Short: "Query accounts",
	Long:  `Query accounts.`,
}

func init() {
	AccountCmd.AddCommand(getCmd)
}
