package backup

import "github.com/spf13/cobra"

// PruneCmd represents the backup command
var PruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Manage pruning",
	Long:  `Manage pruning.`,
}

func init() {
	PruneCmd.AddCommand(executeCmd)
}
