package rpc

import (
	"os"
	"path"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/snapshot"
)

// ------------------------------- BackupSnapshot -----------------------------------

type BackupSnapshotArgs struct {
	Config string `json:"config"`
	Height uint64 `json:"height"`
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

	snapshotFile, err := snapshot.ExportSnapshot(db, consensus, chain, snapshotDir, args.Height)
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
	chain := t.chain
	startHeight := args.Start
	endHeight := args.End

	backupDir := path.Join(args.Config, "backup", "chain")
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		os.MkdirAll(backupDir, os.ModePerm)
	}

	actualStartHeight, actualEndHeight, chainFile, err := snapshot.ExportChainBackup(chain, startHeight, endHeight, backupDir)
	result.ActualStartHeight = actualStartHeight
	result.ActualEndHeight = actualEndHeight
	result.ChainFile = chainFile

	return err
}

// ------------------------------- BackupChainCorrection -----------------------------------

type BackupChainCorrectionArgs struct {
	RollbackHeight uint64      `json:"rollback_height"`
	EndBlockHash   common.Hash `json:"end_block_hash"`
	Config         string      `json:"config"`
}

type BackupChainCorrectionResult struct {
	// StartHeight    uint64 `json:"start_height"`
	// StartBlockHash uint64 `json:"start_block_hash"`
	ChainFile string `json:"chain_correction_file"`
}

func (t *ThetaRPCService) BackupChainCorrection(args *BackupChainCorrectionArgs, result *BackupChainCorrectionResult) error {
	chain := t.chain
	rollbackHeight := args.RollbackHeight
	endBlockHash := args.EndBlockHash

	backupDir := path.Join(args.Config, "backup", "chain_correction")
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		os.MkdirAll(backupDir, os.ModePerm)
	}

	chainFile, err := snapshot.ExportChainCorrection(chain, rollbackHeight, endBlockHash, backupDir)
	// result.StartHeight = actualStartHeight
	// result.StartBlockHash = startBlockHash
	result.ChainFile = chainFile

	return err
}
