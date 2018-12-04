package rpc

import (
	"net/http"
	"time"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/consensus"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/ledger/state"
)

type GenSnapshotArgs struct {
}

type GenSnapshotResult struct {
}

func (t *ThetaRPCServer) GenSnapshot(r *http.Request, args *GenSnapshotArgs, result *GenSnapshotResult) (err error) {
	s := t.consensus.GetSummary()
	latestFinalizedTreeRoot := s.Root
	if !latestFinalizedTreeRoot.IsEmpty() {
		block := &core.Checkpoint{}
		sv := state.NewStoreView(s.LastVoteHeight, latestFinalizedTreeRoot, t.ledger.State().DB())
		sv.GetStore().Traverse(nil, func(k, v common.Bytes) bool {
			block.LedgerState = append(block.LedgerState, core.KVPair{Key: k, Value: v})
			return true
		})
		currentTime := time.Now().UTC()
		err = consensus.WriteCheckpoint("theta_snapshot-"+currentTime.Format("2006-01-02"), block)
	}
	return
}
