package query

import (
	"github.com/spf13/cobra"
)

var (
	purposeFlag          uint8
	heightFlag           uint64
	addressFlag          string
	previewFlag          bool
	resourceIDFlag       string
	hashFlag             string
	startFlag            uint64
	endFlag              uint64
	skipEdgeNodeFlag     bool
	includeEthTxHashFlag bool
	sourceFlag           string
	holderFlag           string
	withdrawnOnlyFlag    bool
)

// QueryCmd represents the query command
var QueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query entities stored in blockchain",
}

func init() {
	QueryCmd.AddCommand(statusCmd)
	QueryCmd.AddCommand(accountCmd)
	QueryCmd.AddCommand(guardianCmd)
	QueryCmd.AddCommand(blockCmd)
	QueryCmd.AddCommand(txCmd)
	QueryCmd.AddCommand(splitRuleCmd)
	QueryCmd.AddCommand(vcpCmd)
	QueryCmd.AddCommand(gcpCmd)
	QueryCmd.AddCommand(eenpCmd)
	QueryCmd.AddCommand(srdrsCmd)
	QueryCmd.AddCommand(stakeReturnsCmd)
	QueryCmd.AddCommand(peersCmd)
	QueryCmd.AddCommand(versionCmd)
}
