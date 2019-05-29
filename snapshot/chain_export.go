package snapshot

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/thetatoken/theta/blockchain"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/rlp"
)

func ExportChainBackup(chain *blockchain.Chain, startHeight, endHeight uint64, backupDir string) (actualStartHeight, actualEndHeight uint64, backupFile string, err error) {
	if startHeight > endHeight {
		return 0, 0, "", errors.New("start height must be <= end height")
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
		return 0, 0, "", fmt.Errorf("There's no finalized block between height %v and %v", startHeight, endHeight)
	}

	currentTime := time.Now().UTC()
	filename := "theta_chain-" + strconv.FormatUint(startHeight, 10) + "-" + strconv.FormatUint(finalizedBlock.Height, 10) + "-" + currentTime.Format("2006-01-02")
	backupPath := path.Join(backupDir, filename)
	file, err := os.Create(backupPath)
	if err != nil {
		return 0, 0, "", err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)

	actualEndHeight = finalizedBlock.Height

	for {
		voteSet := chain.FindVotesByHash(finalizedBlock.Hash())
		backupBlock := &core.BackupBlock{Block: finalizedBlock, Votes: voteSet}
		writeBlock(writer, backupBlock)

		if finalizedBlock.Height <= startHeight {
			break
		}
		parentBlock, err := chain.FindBlock(finalizedBlock.Parent)
		if err != nil {
			filename = "theta_chain-" + strconv.FormatUint(finalizedBlock.Height, 10) + "-" + strconv.FormatUint(actualEndHeight, 10) + "-" + currentTime.Format("2006-01-02")
			actualBackupPath := path.Join(backupDir, filename)
			os.Rename(backupPath, actualBackupPath)
			return finalizedBlock.Height, actualEndHeight, filename, nil
		}
		finalizedBlock = parentBlock
	}

	return startHeight, actualEndHeight, filename, nil
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
