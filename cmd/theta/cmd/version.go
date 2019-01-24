package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thetatoken/theta/version"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version of current Theta binary.",
	Run:   runVersion,
}

func init() {
	RootCmd.AddCommand(versionCmd)
}

func runVersion(cmd *cobra.Command, args []string) {
	fmt.Printf("Version %s\nBuilt at %s\n", version.GitHash, version.Timestamp)
}
