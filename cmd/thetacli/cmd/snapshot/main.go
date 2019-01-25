package snapshot

import "github.com/spf13/cobra"

// SnapshotCmd represents the snapshot command
var SnapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Manage snapshots",
	Long:  `Manage snapshots.`,
}

func init() {
	SnapshotCmd.AddCommand(exportCmd)
}
