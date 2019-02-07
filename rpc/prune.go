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

	if end >= lastFinalizedBlock.Height {
		return fmt.Errorf("Can't prune at height >= %v yet", lastFinalizedBlock.Height)
	}

	stateHashMap := make(map[string]bool)
	for height := end; height >= start && height > 0; height-- {
		blocks := chain.FindBlocksByHeight(height)
		logger.Errorf("===== # blocks at height %v: %v", height, len(blocks))
		for _, block := range blocks {
			logger.Errorf("==============> %v, %v, %v", height, block.StateHash.String(), block.HasValidatorUpdate)
			if _, ok := stateHashMap[block.StateHash.String()]; !ok {
				if block.HasValidatorUpdate {
					continue
				}
				sv := state.NewStoreView(height, block.StateHash, db)
				err = sv.Prune()
				if err != nil {
					return fmt.Errorf("Failed to prune storeview at height %v, %v", height, err)
				}
				// stateHashMap[block.StateHash.String()] = true
				logger.Errorf("==============+++ %v", stateHashMap)
			}
		}
	}
	logger.Errorf("========------------======")
	blocks := chain.FindBlocksByHeight(0)
	logger.Errorf("===== # blocks at height 0: %v", len(blocks))
	for _, block := range blocks {
		logger.Errorf("==============> %v, %v", block.StateHash.String(), block.HasValidatorUpdate)
	}

	return nil
}
