package query

import (
	"github.com/spf13/cobra"
)

// QueryCmd represents the query command
var QueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query entiteis stored in blockchain",
}

func init() {
	QueryCmd.AddCommand(accountCmd)
	QueryCmd.AddCommand(splitRuleCmd)
}
