package key

import (
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"
	"github.com/thetatoken/ukulele/crypto"
)

// newCmd represents the new command
var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Generates a new private key",
	Long:  `Generates a new private key.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfgPath := cmd.Flag("config").Value.String()
		if privKey, err := createNewKey(cfgPath); err != nil {
			fmt.Printf("Failed to generate new key: %v\n", err)
		} else {
			fmt.Printf("Successfully created key: %v\n", privKey.PublicKey().Address())
		}
	},
}

func createNewKey(cfgPath string) (*crypto.PrivateKey, error) {
	privKey, pubKey, err := crypto.GenerateKeyPair()
	if err != nil {
		return nil, err
	}
	dirPath := path.Join(cfgPath, "keys")
	err = os.MkdirAll(dirPath, 0700)

	fi, err := os.Lstat(dirPath)
	if err != nil {
		return nil, err
	}
	if fi.Mode().Perm() != 0700 {
		return nil, fmt.Errorf("%s must have permission set to 0700", dirPath)
	}

	filePath := path.Join(dirPath, pubKey.Address().Hex())
	err = privKey.SaveToFile(filePath)
	return privKey, err
}
