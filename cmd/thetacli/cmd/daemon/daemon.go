package daemon

import (
	"context"
	"log"
	"sync"

	"github.com/spf13/cobra"
	"github.com/thetatoken/theta/cmd/thetacli/rpc"
)

// startDaemonCmd runs the thetacli daemon
// Example:
//		thetacli daemon start --port=16889
var startDaemonCmd = &cobra.Command{
	Use:     "start",
	Short:   "Run the thatacli daemon",
	Long:    `Run the thatacli daemon.`,
	Example: `thetacli daemon start --port=16889`,
	Run: func(cmd *cobra.Command, args []string) {
		cfgPath := cmd.Flag("config").Value.String()
		server, err := rpc.NewThetaCliRPCServer(cfgPath, portFlag)
		if err != nil {
			log.Fatalf("Failed to run the ThetaCli Daemon: %v", err)
		}
		daemon := &ThetaCliDaemon{
			RPC: server,
		}
		daemon.Start(context.Background())
		daemon.Wait()
	},
}

func init() {
	startDaemonCmd.Flags().StringVar(&portFlag, "port", "16889", "Port to run the ThetaCli Daemon")
}

type ThetaCliDaemon struct {
	RPC *rpc.ThetaCliRPCServer

	// Life cycle
	wg      *sync.WaitGroup
	quit    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
}

func (d *ThetaCliDaemon) Start(ctx context.Context) {
	c, cancel := context.WithCancel(ctx)
	d.ctx = c
	d.cancel = cancel

	if d.RPC != nil {
		d.RPC.Start(d.ctx)
	}
}

func (d *ThetaCliDaemon) Stop() {
	d.cancel()
}

func (d *ThetaCliDaemon) Wait() {
	if d.RPC != nil {
		d.RPC.Wait()
	}
}
