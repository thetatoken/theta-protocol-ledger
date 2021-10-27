package rpc

import (
	"os"
	"path"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/snapshot"
)

// ------------------------------- BackupSnapshot -----------------------------------

type BackupSnapshotArgs struct {
	Config  string `json:"config"`
	Height  uint64 `json:"height"`
	Version uint64 `json:"version"`
}

type BackupSnapshotResult struct {
	SnapshotFile string `json:"snapshot_file"`
}

func (t *ThetaRPCService) BackupSnapshot(args *BackupSnapshotArgs, result *BackupSnapshotResult) error {
	// Default to older verison
	if args.Version == 0 {
		args.Version = 2
	}

	db := t.ledger.State().DB()
	consensus := t.consensus
	chain := t.chain

	snapshotDir := path.Join(args.Config, "backup", "snapshot")
	if _, err := os.Stat(snapshotDir); os.IsNotExist(err) {
		os.MkdirAll(snapshotDir, os.ModePerm)
	}

	if args.Version == 2 {
		snapshotFile, err := snapshot.ExportSnapshotV2(db, consensus, chain, snapshotDir, args.Height)
		result.SnapshotFile = snapshotFile
		return err
	} else if args.Version == 3 {
		snapshotFile, err := snapshot.ExportSnapshotV3(db, consensus, chain, snapshotDir, args.Height)
		result.SnapshotFile = snapshotFile
		return err
	}

	snapshotFile, err := snapshot.ExportSnapshotV4(db, consensus, chain, snapshotDir, args.Height)
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
	SnapshotHeight uint64      `json:"snapshot_height"`
	EndBlockHash   common.Hash `json:"end_block_hash"`
	Config         string      `json:"config"`
	ExclusionTxs   []string    `json:"exclusion_txs"`
}

type BackupChainCorrectionResult struct {
	ChainFile    string            `json:"chain_correction_file"`
	BlockHashMap map[uint64]string `json:"block_hash_map"`
}

func (t *ThetaRPCService) BackupChainCorrection(args *BackupChainCorrectionArgs, result *BackupChainCorrectionResult) error {
	chain := t.chain
	ledger := t.consensus.GetLedger()
	snapshotHeight := args.SnapshotHeight
	endBlockHash := args.EndBlockHash
	exclusionTxs := args.ExclusionTxs

	backupDir := path.Join(args.Config, "backup", "chain_correction")
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		os.MkdirAll(backupDir, os.ModePerm)
	}

	chainFile, blockHashMap, err := snapshot.ExportChainCorrection(chain, ledger, snapshotHeight, endBlockHash, backupDir, exclusionTxs)
	result.ChainFile = chainFile
	result.BlockHashMap = blockHashMap

	return err
}
