package rpc

import (
	"os"
	"path"

	"github.com/thetatoken/theta/snapshot"
)

// ------------------------------- BackupSnapshot -----------------------------------

type BackupSnapshotArgs struct {
	Config string `json:"config"`
}

type BackupSnapshotResult struct {
	SnapshotFile string `json:"snapshot_file"`
}

func (t *ThetaRPCService) BackupSnapshot(args *BackupSnapshotArgs, result *BackupSnapshotResult) error {
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

// ------------------------------- BackupChain -----------------------------------

type BackupChainArgs struct {
	Start  uint64 `json:"start"`
	End    uint64 `json:"end"`
	Config string `json:"config"`
}

type BackupChainResult struct {
	ActualStartHeight uint64 `json:"actual_start_height"`
	ActualEndHeight   uint64 `json:"actual_end_height"`
	ChainFile         string `json:"chain_file"`
}

func (t *ThetaRPCService) BackupChain(args *BackupChainArgs, result *BackupChainResult) error {
	db := t.ledger.State().DB()
	consensus := t.consensus
	chain := t.chain
	startHeight := args.Start
	endHeight := args.End

	backupDir := path.Join(args.Config, "backup", "chain")
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		os.MkdirAll(backupDir, os.ModePerm)
	}

	actualStartHeight, actualEndHeight, chainFile, err := snapshot.ExportChainBackup(db, consensus, chain, startHeight, endHeight, backupDir)
	result.ActualStartHeight = actualStartHeight
	result.ActualEndHeight = actualEndHeight
	result.ChainFile = chainFile

	return err
}
