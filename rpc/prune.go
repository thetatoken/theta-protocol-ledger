package rpc

import (
	"fmt"

	"github.com/thetatoken/theta/blockchain"
	cns "github.com/thetatoken/theta/consensus"
	"github.com/thetatoken/theta/ledger/state"
	"github.com/thetatoken/theta/store/database"
)

type PruneArgs struct {
	Start uint64 `json:"start"`
	End   uint64 `json:"end"`
}

type PruneResult struct {
}

func (t *ThetaRPCService) ExecutePrune(args *PruneArgs, result *PruneResult) error {
	db := t.ledger.State().DB()
	consensus := t.consensus
	chain := t.chain
	start := args.Start
	end := args.End

	return prune(start, end, db, consensus, chain)
}

func prune(start uint64, end uint64, db database.Database, consensus *cns.ConsensusEngine, chain *blockchain.Chain) error {
	stub := consensus.GetSummary()
	lastFinalizedBlock, err := chain.FindBlock(stub.LastFinalizedBlock)
	if err != nil {
		return fmt.Errorf("Failed to get last finalized block %v, %v", stub.LastFinalizedBlock, err)
	}

	if end > lastFinalizedBlock.Height {
		return fmt.Errorf("Can't prune at height > %v yet", lastFinalizedBlock.Height)
	}

	for height := end; height >= start; height-- {
		blocks := chain.FindBlocksByHeight(height)
		for _, block := range blocks {
			if block.HasValidatorUpdate {
				continue
			}
			sv := state.NewStoreView(height, block.StateHash, db)
			err = sv.Prune()
			if err != nil {
				return fmt.Errorf("Failed to prune storeview at height %v, %v", height, err)
			}
		}
	}
	return nil
}
