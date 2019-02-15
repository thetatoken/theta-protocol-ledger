package backup

import "github.com/spf13/cobra"

// BackupCmd represents the backup command
var BackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage backups",
	Long:  `Manage backups.`,
}

func init() {
	BackupCmd.AddCommand(chainCmd)
	BackupCmd.AddCommand(snapshotCmd)
}
