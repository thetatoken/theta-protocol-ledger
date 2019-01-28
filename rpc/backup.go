package rpc

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/thetatoken/theta/consensus"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/rlp"
	"github.com/thetatoken/theta/store/kvstore"
)

type BackupArgs struct {
	Start uint64 `json:"start"`
	End   uint64 `json:"end"`
}

type BackupResult struct {
	ActualEndHeight uint64 `json:"actual_end_height"`
}

func (t *ThetaRPCService) GenBackup(args *BackupArgs, result *BackupResult) error {
	startHeight := args.Start
	endHeight := args.End

	if startHeight > endHeight {
		return errors.New("start height must be <= end height")
	}

	var finalizedBlock *core.ExtendedBlock
	for i := endHeight; i >= startHeight; i-- {
		blocks := t.chain.FindBlocksByHeight(i)
		for _, block := range blocks {
			if block.Status.IsFinalized() {
				finalizedBlock = block
				break
			}
		}
		if finalizedBlock != nil {
			break
		}
	}

	if finalizedBlock == nil {
		return fmt.Errorf("There's no finalized block between height %v and %v", startHeight, endHeight)
	}

	currentTime := time.Now().UTC()
	file, err := os.Create("theta_backup-" + strconv.FormatUint(startHeight, 64) + "-" + strconv.FormatUint(finalizedBlock.Height, 64) + "-" + currentTime.Format("2006-01-02"))
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	db := t.ledger.State().DB()
	st := consensus.NewState(kvstore.NewKVStore(db), t.chain)

	for {
		voteSet, err := st.GetVoteSetByBlock(finalizedBlock.Hash())
		if err != nil {
			return fmt.Errorf("Failed to get block's voteset, %v", err)
		}
		backupBlock := &core.BackupBlock{Block: finalizedBlock, Votes: voteSet}
		writeBlock(writer, backupBlock)

		if finalizedBlock.Height <= startHeight || finalizedBlock.Height <= 0 {
			break
		}
		finalizedBlock, err := t.chain.FindBlock(finalizedBlock.Parent)
		if err != nil {
			return fmt.Errorf("Failed to get parent block %v, %v", finalizedBlock.Parent, err)
		}
	}

	result.ActualEndHeight = finalizedBlock.Height
	return nil
}

func writeBlock(writer *bufio.Writer, block *core.BackupBlock) error {
	raw, err := rlp.EncodeToBytes(*block)
	if err != nil {
		log.Error("Failed to encode backup block")
		return err
	}
	// write length first
	_, err = writer.Write(core.Itobytes(uint64(len(raw))))
	if err != nil {
		log.Error("Failed to write backup block length")
		return err
	}
	// write metadata itself
	_, err = writer.Write(raw)
	if err != nil {
		log.Error("Failed to write backup block")
		return err
	}
	return nil
}
