package rpc

import (
	"github.com/thetatoken/theta/snapshot"
)

type BackupArgs struct {
	Start uint64 `json:"start"`
	End   uint64 `json:"end"`
}

type BackupResult struct {
	ActualEndHeight uint64 `json:"actual_end_height"`
}

func (t *ThetaRPCService) GenBackup(args *BackupArgs, result *BackupResult) error {
	db := t.ledger.State().DB()
	consensus := t.consensus
	chain := t.chain
	startHeight := args.Start
	endHeight := args.End

	actualEndHeight, err := snapshot.ExportChainBackup(db, consensus, chain, startHeight, endHeight)
	result.ActualEndHeight = actualEndHeight

	return err
}
