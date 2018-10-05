package key

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/spf13/cobra"
)

// listCmd represents the new command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all keys",
	Long:  `List all keys.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfgPath := cmd.Flag("config").Value.String()
		if filenames, err := listKeys(cfgPath); err != nil {
			fmt.Printf("Failed to list keys: %v\n", err)
		} else {
			for _, filename := range filenames {
				fmt.Printf("%s\n", filepath.Base(filename))
			}
		}
	},
}

func listKeys(cfgPath string) ([]string, error) {
	dirPath := path.Join(cfgPath, "keys")
	return filepath.Glob(path.Join(dirPath, "*"))
}
