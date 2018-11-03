package call

import (
	"github.com/spf13/cobra"
)

// Common flags used in Call sub commands.
var (
	chainIDFlag  string
	fromFlag     string
	toFlag       string
	seqFlag      uint64
	valueFlag    uint64
	gasPriceFlag uint64
	gasLimitFlag uint64
	dataFlag     string
)

// CallCmd represents the call command
var CallCmd = &cobra.Command{
	Use:   "call",
	Short: "Call smart contract APIs",
}

func init() {
	CallCmd.AddCommand(smartContractCmd)
}
