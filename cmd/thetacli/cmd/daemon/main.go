package daemon

import (
	"github.com/spf13/cobra"
)

var (
	portFlag string
)

// DaemonCmd represents the call command
var DaemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run the ThetaCli Daemon",
}

func init() {
	DaemonCmd.AddCommand(startDaemonCmd)
}
