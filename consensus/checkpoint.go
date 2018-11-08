package consensus

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/thetatoken/ukulele/crypto"
	"github.com/thetatoken/ukulele/ledger/state"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/rlp"
	"github.com/thetatoken/ukulele/store/database"
	"github.com/thetatoken/ukulele/store/database/backend"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/core"
)

// LoadCheckpoint loads a checkpoint from file system.
func LoadCheckpoint(filePath string) (*core.Checkpoint, error) {
	r, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	checkpoint := &core.Checkpoint{}
	err = rlp.Decode(r, checkpoint)
	return checkpoint, err
}

// LoadCheckpointLedgerState insert LedgerState entries in checkpoint into database.
func LoadCheckpointLedgerState(checkpoint *core.Checkpoint, db database.Database) {
	s := state.NewStoreView(0, common.Hash{}, db)
	for _, pair := range checkpoint.LedgerState {
		s.Set(pair.Key, pair.Value)
	}
	actualHash := s.Save()
	if actualHash != checkpoint.FirstBlock.StateHash {
		panic(fmt.Sprintf("Acutal hash %v != expected hash: %v", actualHash, checkpoint.FirstBlock.StateHash))
	}
}

// WriteGenesisCheckpoint writes genesis checkpoint to file system.
func WriteGenesisCheckpoint(filePath string) error {
	genesis, err := generateGenesisCheckpoint()
	if err != nil {
		return err
	}
	return WriteCheckpoint(filePath, genesis)
}

// WriteCheckpoint writes a checkpoint to file system.
func WriteCheckpoint(filePath string, checkpoint *core.Checkpoint) error {
	raw, err := rlp.EncodeToBytes(checkpoint)
	if err != nil {
		return err
	}
	return common.WriteFileAtomic(filePath, raw, 0600)
}

// generateGenesisCheckpoint generates the genesis checkpoint.
func generateGenesisCheckpoint() (*core.Checkpoint, error) {
	genesis := &core.Checkpoint{}

	genesis.Validators = []string{
		"042CA7FFB62122A220C72AA7CD87C252B21D72273275682386A099F0983C135659FF93E2E8756011074706E18113AA6529CD5833DD6463266980C6973895153C7C",
		"048E8D53FD435265AD074597CC3E202F8E935CFB57925BB51316252027CB08767FB8099226414732543C4B5CBAA64B4EE8F173BA559258A0B5F633A0D11509E78B",
		"0479188733862EBB3FE98A92315556D5214D908941CDC8D6C8700EEEAE5F90A6177A37E23B33B81B9FAC3A98EE2382AB24B1C92384FC151D07E36AC7209702D353",
		"0455BDC5CF697F9519DF40E837BEE3E246C8D47C1B58CD1892FD3B0F780D2C09E718FF50A5929B86B8B88C7031164BDE553E285103F1B4DF668B44AFC907264C1C",
	}

	s := state.NewStoreView(0, common.Hash{}, backend.NewMemDatabase())
	for _, v := range genesis.Validators {
		raw, err := hex.DecodeString(v)
		if err != nil {
			return nil, err
		}
		pubKey, err := crypto.PublicKeyFromBytes(raw)
		if err != nil {
			return nil, err
		}
		acc := &types.Account{
			PubKey: pubKey,
			Balance: types.Coins{
				ThetaWei: new(big.Int).SetUint64(1e15),
				GammaWei: new(big.Int).SetUint64(1e15),
			},
			LastUpdatedBlockHeight: 0,
		}
		s.SetAccount(acc.PubKey.Address(), acc)
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
