package key

import (
	"fmt"

	"github.com/spf13/cobra"
)

// recoverCmd recovers the key from the given seed phrase
var recoverCmd = &cobra.Command{
	Use:   "recover",
	Short: "Recover a key from seed phrase",
	Long:  `Recover a key from seed phrase.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("key recover not supported yet")
	},
}
