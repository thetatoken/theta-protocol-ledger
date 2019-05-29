package snapshot

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/thetatoken/theta/blockchain"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/core"
)

type BHStack []common.Hash

func (stack BHStack) push(hash common.Hash) BHStack {
	return append(stack, hash)
}

func (stack BHStack) pop() (BHStack, common.Hash) {
	l := len(stack)
	if l == 0 {
		return stack, common.Hash{}
	}
	return stack[:l-1], stack[l-1]
}

// func (stack BHStack) peek() common.Hash {
// 	l := len(stack)
// 	if l == 0 {
// 		return common.Hash{}
// 	}
// 	return stack[l-1]
// }

func ExportChainCorrection(chain *blockchain.Chain, rollbackHeight uint64, endBlockHash common.Hash, backupDir string) (backupFile string, err error) {
	block, err := chain.FindBlock(endBlockHash)
	if err != nil {
		return "", fmt.Errorf("Can't find block for hash %v", endBlockHash)
	}

	if rollbackHeight >= block.Height {
		return "", errors.New("Start height must be < end height")
	}

	filename := "theta_chain_correction-" + strconv.FormatUint(rollbackHeight, 10) + "-" + strconv.FormatUint(block.Height, 10)
	backupPath := path.Join(backupDir, filename)
	file, err := os.Create(backupPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)

	bhStack := make(BHStack, 0)
	bhStackRev := make(BHStack, 0)

	for {
		block.TxHash = core.CalculateRootHash(block.Txs) //TODO: add functionality to exclude certian tx
		bh := block.UpdateHash()
		bhStack = bhStack.push(bh)

		if block.Height <= rollbackHeight {
			break
		}
		parentBlock, err := chain.FindBlock(block.Parent)
		if err != nil {
			return "", fmt.Errorf("Can't find block for %v", block.Hash())
		}
		block = parentBlock
	}

	var bh common.Hash
	parentBH := common.Hash{}
	for {
		bhStack, bh = bhStack.pop()

		if (bh == common.Hash{}) {
			break
		}

		if (parentBH != common.Hash{}) {
			block, _ := chain.FindBlock(bh)
			block.Parent = parentBH
			bh = block.UpdateHash()
		}
		parentBH = bh

		bhStackRev = bhStackRev.push(bh)
	}

	for {
		bhStackRev, bh = bhStackRev.pop()

		if (bh == common.Hash{}) {
			break
		}

		backupBlock := &core.BackupBlock{Block: block}
		writeBlock(writer, backupBlock)
	}

	return filename, nil
}
