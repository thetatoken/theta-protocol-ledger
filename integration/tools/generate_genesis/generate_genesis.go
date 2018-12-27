package main

import (
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

// Example: generate_genesis -erc20snapshot=./theta_erc20_snapshot.json -genesis=../testnet/node2/genesis
func main() {
	erc20SnapshotJsonFilePathPtr := flag.String("erc20snapshot", "./theta_erc20_snapshot.json", "the json file contain the ERC20 balance snapshot")
	genesisCheckpointfilePathPtr := flag.String("genesis", "./genesis", "the genesis checkpoint")
	flag.Parse()

	erc20SnapshotJsonFilePath := *erc20SnapshotJsonFilePathPtr
	genesisCheckpointfilePath := *genesisCheckpointfilePathPtr

	writeGenesisCheckpoint(erc20SnapshotJsonFilePath, genesisCheckpointfilePath)
}

// writeGenesisCheckpoint writes genesis checkpoint to file system.
func writeGenesisCheckpoint(erc20SnapshotJsonFilePath, genesisCheckpointfilePath string) error {
	genesis, err := generateGenesisCheckpoint(erc20SnapshotJsonFilePath)
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
func generateGenesisCheckpoint(erc20SnapshotJsonFilePath string) (*core.Checkpoint, error) {
	genesis := &core.Checkpoint{}

	genesis.Validators = []string{
		"042CA7FFB62122A220C72AA7CD87C252B21D72273275682386A099F0983C135659FF93E2E8756011074706E18113AA6529CD5833DD6463266980C6973895153C7C",
		"048E8D53FD435265AD074597CC3E202F8E935CFB57925BB51316252027CB08767FB8099226414732543C4B5CBAA64B4EE8F173BA559258A0B5F633A0D11509E78B",
		"0479188733862EBB3FE98A92315556D5214D908941CDC8D6C8700EEEAE5F90A6177A37E23B33B81B9FAC3A98EE2382AB24B1C92384FC151D07E36AC7209702D353",
		"0455BDC5CF697F9519DF40E837BEE3E246C8D47C1B58CD1892FD3B0F780D2C09E718FF50A5929B86B8B88C7031164BDE553E285103F1B4DF668B44AFC907264C1C",
	}

	initGammaToThetaRatio := new(big.Int).SetUint64(5)
	s := state.NewStoreView(0, common.Hash{}, backend.NewMemDatabase())

	// Load initial balances
	erc20SnapshotJsonFile, err := os.Open(erc20SnapshotJsonFilePath)
	if err != nil {
		panic(fmt.Sprintf("failed to open the ERC20 balance snapshot: %v", err))
	}
	defer erc20SnapshotJsonFile.Close()

	var erc20BalanceMap map[string]string
	byteValue, err := ioutil.ReadAll(erc20SnapshotJsonFile)
	if err != nil {
		panic(fmt.Sprintf("failed to read the ERC20 balance snapshot: %v", err))
	}

	json.Unmarshal([]byte(byteValue), &erc20BalanceMap)

	for key, val := range erc20BalanceMap {
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
		return false
	})

	return genesis, nil
}
