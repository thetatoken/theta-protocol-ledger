package backup

import "github.com/spf13/cobra"

var (
	heightFlag uint64
	hashFlag   string
	configFlag string
)

// BackupCmd represents the backup command
var BackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage backups",
	Long:  `Manage backups.`,
}

func init() {
	BackupCmd.AddCommand(chainCmd)
	BackupCmd.AddCommand(snapshotCmd)
	BackupCmd.AddCommand(chainCorrectionCmd)
}
