package call

import (
	"github.com/spf13/cobra"
)

// Common flags used in Call sub commands.
var (
	fromFlag     string
	toFlag       string
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
