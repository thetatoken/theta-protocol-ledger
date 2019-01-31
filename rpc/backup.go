package rpc

import (
	"os"
	"path"

	"github.com/thetatoken/theta/snapshot"
)

type BackupArgs struct {
	Start  uint64 `json:"start"`
	End    uint64 `json:"end"`
	Config string `json:"config"`
}

type BackupResult struct {
	ActualEndHeight uint64 `json:"actual_end_height"`
	BackupFile      string `json:"backup_file"`
}

func (t *ThetaRPCService) GenBackup(args *BackupArgs, result *BackupResult) error {
	db := t.ledger.State().DB()
	consensus := t.consensus
	chain := t.chain
	startHeight := args.Start
	endHeight := args.End

	backupDir := path.Join(args.Config, "backup", "chain")
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		os.MkdirAll(backupDir, os.ModePerm)
	}

	actualEndHeight, backupFile, err := snapshot.ExportChainBackup(db, consensus, chain, startHeight, endHeight, backupDir)
	result.ActualEndHeight = actualEndHeight
	result.BackupFile = backupFile

	return err
}
