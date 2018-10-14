package tx

import (
	"path"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thetatoken/ukulele/crypto"
)

// Common flags used in Tx sub commands.
var (
	chainIDFlag                  string
	fromFlag                     string
	toFlag                       string
	seqFlag                      uint64
	thetaAmountFlag              int64
	gammaAmountFlag              int64
	gasAmountFlag                uint64
	feeInGammaFlag               int64
	resourceIDsFlag              []string
	resourceIDFlag               string
	durationFlag                 uint64
	reserveFundInGammaFlag       int64
	reserveCollateralInGammaFlag int64
	reserveSeqFlag               uint64
	addressesFlag                []string
	percentagesFlag              []string
)

// TxCmd represents the Tx command
var TxCmd = &cobra.Command{
	Use:   "tx",
	Short: "Manage transactions",
	Long:  `Manage transactions.`,
}

func init() {
	TxCmd.AddCommand(sendCmd)
	TxCmd.AddCommand(reserveFundCmd)
	TxCmd.AddCommand(releaseFundCmd)
	TxCmd.AddCommand(splitContractCmd)
}

func loadPrivateKey(cfgPath string, address string) (*crypto.PrivateKey, error) {
	if strings.HasPrefix(address, "0x") {
		address = address[2:]
	}
	filePath := path.Join(cfgPath, "keys", address)
	return crypto.PrivateKeyFromFile(filePath)
}
