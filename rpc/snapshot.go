package rpc

import (
	"github.com/thetatoken/theta/snapshot"
)

type GenSnapshotArgs struct {
}

type GenSnapshotResult struct {
}

func (t *ThetaRPCService) GenSnapshot(args *GenSnapshotArgs, result *GenSnapshotResult) error {
	db := t.ledger.State().DB()
	consensus := t.consensus
	chain := t.chain
	err := snapshot.ExportSnapshot(db, consensus, chain)

	return err
}
