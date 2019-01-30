package snapshot

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/thetatoken/theta/blockchain"
	cns "github.com/thetatoken/theta/consensus"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/rlp"
	"github.com/thetatoken/theta/store/database"
	"github.com/thetatoken/theta/store/kvstore"
)

func ExportChainBackup(db database.Database, consensus *cns.ConsensusEngine, chain *blockchain.Chain, startHeight, endHeight uint64) (actualEndHeight uint64, err error) {
	if startHeight > endHeight {
		return 0, errors.New("start height must be <= end height")
	}

	var finalizedBlock *core.ExtendedBlock
	for i := endHeight; i >= startHeight; i-- {
		blocks := chain.FindBlocksByHeight(i)
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
		return 0, fmt.Errorf("There's no finalized block between height %v and %v", startHeight, endHeight)
	}

	currentTime := time.Now().UTC()
	file, err := os.Create("theta_backup-" + strconv.FormatUint(startHeight, 10) + "-" + strconv.FormatUint(finalizedBlock.Height, 10) + "-" + currentTime.Format("2006-01-02"))
	if err != nil {
		return 0, err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	st := cns.NewState(kvstore.NewKVStore(db), chain)

	actualEndHeight = finalizedBlock.Height

	for {
		voteSet, err := st.GetVoteSetByBlock(finalizedBlock.Hash())
		if err != nil {
			return 0, fmt.Errorf("Failed to get block's voteset, %v", err)
		}
		backupBlock := &core.BackupBlock{Block: finalizedBlock, Votes: voteSet}
		writeBlock(writer, backupBlock)

		if finalizedBlock.Height <= startHeight {
			break
		}
		finalizedBlock, err = chain.FindBlock(finalizedBlock.Parent)
		if err != nil {
			return 0, fmt.Errorf("Failed to get parent block %v, %v", finalizedBlock.Parent, err)
		}
	}

	return actualEndHeight, nil
}

func writeBlock(writer *bufio.Writer, block *core.BackupBlock) error {
	raw, err := rlp.EncodeToBytes(*block)
	if err != nil {
		logger.Error("Failed to encode backup block")
		return err
	}
	// write length first
	_, err = writer.Write(core.Itobytes(uint64(len(raw))))
	if err != nil {
		logger.Error("Failed to write backup block length")
		return err
	}
	// write metadata itself
	_, err = writer.Write(raw)
	if err != nil {
		logger.Error("Failed to write backup block")
		return err
	}
	writer.Flush()
	return nil
}
