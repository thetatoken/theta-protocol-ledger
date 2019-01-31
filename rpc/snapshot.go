package rpc

import (
	"os"
	"path"

	"github.com/thetatoken/theta/snapshot"
)

type GenSnapshotArgs struct {
	Config string `json:"config"`
}

type GenSnapshotResult struct {
	SnapshotFile string `json:"snapshot_file"`
}

func (t *ThetaRPCService) GenSnapshot(args *GenSnapshotArgs, result *GenSnapshotResult) error {
	db := t.ledger.State().DB()
	consensus := t.consensus
	chain := t.chain

	snapshotDir := path.Join(args.Config, "backup", "snapshot")
	if _, err := os.Stat(snapshotDir); os.IsNotExist(err) {
		os.MkdirAll(snapshotDir, os.ModePerm)
	}

	snapshotFile, err := snapshot.ExportSnapshot(db, consensus, chain, snapshotDir)
	result.SnapshotFile = snapshotFile

	return err
}
