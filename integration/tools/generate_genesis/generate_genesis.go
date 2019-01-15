package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/ledger/state"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/rlp"
	"github.com/thetatoken/ukulele/store/database/backend"
)

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "genesis"})

// Example: generate_genesis -erc20snapshot=./theta_erc20_snapshot.json -stake_deposit=./stake_deposit.json -genesis=../testnet/node2/genesis
func main() {
	erc20SnapshotJsonFilePathPtr := flag.String("erc20snapshot", "./theta_erc20_snapshot.json", "the json file contain the ERC20 balance snapshot")
	stakeDepositFilePathPtr := flag.String("stake_deposit", "./stake_deposit.json", "the initial stake deposits")
	genesisCheckpointfilePathPtr := flag.String("genesis", "./genesis", "the genesis checkpoint")
	flag.Parse()

	erc20SnapshotJsonFilePath := *erc20SnapshotJsonFilePathPtr
	stakeDepositFilePath := *stakeDepositFilePathPtr
	genesisCheckpointfilePath := *genesisCheckpointfilePathPtr

	writeGenesisCheckpoint(erc20SnapshotJsonFilePath, stakeDepositFilePath, genesisCheckpointfilePath)
}

type StakeDeposit struct {
	Source string `json:"source"`
	Holder string `json:"holder"`
	Amount string `json:"amount"`
}

// writeGenesisCheckpoint writes genesis checkpoint to file system.
func writeGenesisCheckpoint(erc20SnapshotJsonFilePath, stakeDepositFPath, genesisCheckpointfilePath string) error {
	genesis, err := generateGenesisCheckpoint(erc20SnapshotJsonFilePath, stakeDepositFPath)
	if err != nil {
		return err
	}

	raw, err := rlp.EncodeToBytes(genesis)
	if err != nil {
		return err
	}
	err = common.WriteFileAtomic(genesisCheckpointfilePath, raw, 0600)
	fmt.Printf("\nGenesis snapshot generated and saved to %v\n\n", genesisCheckpointfilePath)

	return err
}

// generateGenesisCheckpoint generates the genesis checkpoint.
func generateGenesisCheckpoint(erc20SnapshotJsonFilePath, stakeDepositFilePath string) (*core.Checkpoint, error) {
	genesis := &core.Checkpoint{}

	initGammaToThetaRatio := new(big.Int).SetUint64(5)
	s := state.NewStoreView(0, common.Hash{}, backend.NewMemDatabase())

	// --------------- Load initial balances --------------- //

	erc20SnapshotJsonFile, err := os.Open(erc20SnapshotJsonFilePath)
	if err != nil {
		panic(fmt.Sprintf("failed to open the ERC20 balance snapshot: %v", err))
	}
	defer erc20SnapshotJsonFile.Close()

	var erc20BalanceMap map[string]string
	erc20BalanceMapByteValue, err := ioutil.ReadAll(erc20SnapshotJsonFile)
	if err != nil {
		panic(fmt.Sprintf("failed to read the ERC20 balance snapshot: %v", err))
	}

	json.Unmarshal([]byte(erc20BalanceMapByteValue), &erc20BalanceMap)
	for key, val := range erc20BalanceMap {
		if !common.IsHexAddress(key) {
			panic(fmt.Sprintf("Invalid address: %v", key))
		}
		address := common.HexToAddress(key)

		theta, success := new(big.Int).SetString(val, 10)
		if !success {
			panic(fmt.Sprintf("Failed to parse ThetaWei amount: %v", val))
		}
		gamma := new(big.Int).Mul(initGammaToThetaRatio, theta)
		acc := &types.Account{
			Address: address,
			Balance: types.Coins{
				ThetaWei: theta,
				GammaWei: gamma,
			},
			LastUpdatedBlockHeight: 0,
		}
		s.SetAccount(acc.Address, acc)

		//fmt.Println(fmt.Sprintf("address: %v, theta: %v, gamma: %v", strings.ToLower(address.String()), theta, gamma))
	}

	// --------------- Perform initial stake deposit --------------- //

	var stakeDeposits []StakeDeposit
	stakeDepositFile, err := os.Open(stakeDepositFilePath)
	stakeDepositByteValue, err := ioutil.ReadAll(stakeDepositFile)
	if err != nil {
		panic(fmt.Sprintf("failed to read the ERC20 balance snapshot: %v", err))
	}

	json.Unmarshal([]byte(stakeDepositByteValue), &stakeDeposits)
	vcp := &core.ValidatorCandidatePool{}
	for _, stakeDeposit := range stakeDeposits {
		if !common.IsHexAddress(stakeDeposit.Source) {
			panic(fmt.Sprintf("Invalid source address: %v", stakeDeposit.Source))
		}
		if !common.IsHexAddress(stakeDeposit.Holder) {
			panic(fmt.Sprintf("Invalid holder address: %v", stakeDeposit.Holder))
		}
		sourceAddress := common.HexToAddress(stakeDeposit.Source)
		holderAddress := common.HexToAddress(stakeDeposit.Holder)
		stakeAmount, success := new(big.Int).SetString(stakeDeposit.Amount, 10)
		if !success {
			panic(fmt.Sprintf("Failed to parse Stake amount: %v", stakeDeposit.Amount))
		}

		sourceAccount := s.GetAccount(sourceAddress)
		if sourceAccount == nil {
			panic(fmt.Sprintf("Failed to retrieve account for source address: %v", sourceAddress))
		}
		if sourceAccount.Balance.ThetaWei.Cmp(stakeAmount) < 0 {
			panic(fmt.Sprintf("The source account %v does NOT have sufficient balance for stake deposit. ThetaWeiBalance = %v, StakeAmount = %v",
				sourceAddress, sourceAccount.Balance.ThetaWei, stakeDeposit.Amount))
		}
		err := vcp.DepositStake(sourceAddress, holderAddress, stakeAmount)
		if err != nil {
			panic(fmt.Sprintf("Failed to deposit stake, err: %v", err))
		}

		stake := types.Coins{
			ThetaWei: stakeAmount,
			GammaWei: new(big.Int).SetUint64(0),
		}
		sourceAccount.Balance = sourceAccount.Balance.Minus(stake)
		s.SetAccount(sourceAddress, sourceAccount)
	}

	s.UpdateValidatorCandidatePool(vcp)

	stateHash := s.Hash()

	firstBlock := core.NewBlock()
	firstBlock.Height = 0
	firstBlock.Epoch = 0
	firstBlock.Parent = common.Hash{}
	firstBlock.StateHash = stateHash
	firstBlock.Timestamp = big.NewInt(time.Now().Unix())

	genesis.FirstBlock = firstBlock

	s.GetStore().Traverse(nil, func(k, v common.Bytes) bool {
		genesis.LedgerState = append(genesis.LedgerState, core.KVPair{Key: k, Value: v})

		logger.Infof("key: %v, val: %v", string(k), hex.EncodeToString(v))

		return false
	})

	// --------------- Sanity Checks --------------- //

	err = sanityChecks(genesis)
	if err != nil {
		panic(fmt.Sprintf("Sanity checks failed: %v", err))
	}

	return genesis, nil
}

func sanityChecks(genesis *core.Checkpoint) error {
	// Check #1: Sum(ThetaWei) + Sum(Stake) == 1 * 10^9 * 10^18

	// Check #2: Sum(GammaWei) == 5 * 10^9 * 10^18

	return nil
}
