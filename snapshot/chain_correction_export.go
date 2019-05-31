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
	"github.com/thetatoken/theta/ledger/types"
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

func (stack BHStack) isEmpty() bool {
	return len(stack) == 0
}

func ExcludeTxs(txs []common.Bytes, exclusionTxs []string, chain *blockchain.Chain) (results []common.Bytes) {
	for _, tx := range txs {
		raw, _, found := chain.FindTxByHash(common.BytesToHash(tx))
		if found {
			t, err := types.TxFromBytes(raw)
			if err != nil {
				continue
			}

			// exclude stake updating txs as well
			if _, ok := t.(*types.DepositStakeTx); ok {
				continue
			}
			if _, ok := t.(*types.WithdrawStakeTx); ok {
				continue
			}

			hash := common.Bytes2Hex(tx)
			found = false
			for _, exclusion := range exclusionTxs {
				if hash == exclusion {
					found = true
					break
				}
			}
			if !found {
				results = append(results, tx)
			}
		}
	}
	return
}

func ExportChainCorrection(chain *blockchain.Chain, snapshotHeight uint64, endBlockHash common.Hash, backupDir string, exclusionTxs []string) (backupFile, headBlockHash string, err error) {
	block, err := chain.FindBlock(endBlockHash)
	if err != nil {
		return "", "", fmt.Errorf("Can't find block for hash %v", endBlockHash)
	}

	if snapshotHeight >= block.Height {
		return "", "", errors.New("Start height must be < end height")
	}

	backupFile = "theta_chain_correction-" + strconv.FormatUint(snapshotHeight, 10) + "-" + strconv.FormatUint(block.Height, 10)
	backupPath := path.Join(backupDir, backupFile)
	file, err := os.Create(backupPath)
	if err != nil {
		return "", "", err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)

	bhStack := make(BHStack, 0)
	bhStackRev := make(BHStack, 0)

	for {
		block.Txs = ExcludeTxs(block.Txs, exclusionTxs, chain)
		block.TxHash = core.CalculateRootHash(block.Txs)
		bh := block.UpdateHash()
		bhStack = bhStack.push(bh)
		chain.SaveBlock(block)

		if block.Height <= snapshotHeight+1 {
			break
		}
		parentBlock, err := chain.FindBlock(block.Parent)
		if err != nil {
			return "", "", fmt.Errorf("Can't find block for %v", block.Hash())
		}
		block = parentBlock
	}

	var bh common.Hash
	parentBH := common.Hash{}
	for {
		bhStack, bh = bhStack.pop()

		if (parentBH != common.Hash{}) {
			block, _ := chain.FindBlock(bh)
			block.Parent = parentBH
			bh = block.UpdateHash()
		}
		parentBH = bh

		bhStackRev = bhStackRev.push(bh)

		if bhStack.isEmpty() {
			break
		}
	}

	for {
		bhStackRev, bh = bhStackRev.pop()

		block, err := chain.FindBlock(bh)
		if err != nil {
			return "", "", fmt.Errorf("Cannot find block for hash %v", bh)
		}

		backupBlock := &core.BackupBlock{Block: block}
		writeBlock(writer, backupBlock)

		headBlockHash = block.Hash().Hex()

		if bhStackRev.isEmpty() {
			break
		}
	}

	return
}
