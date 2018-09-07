package consensus

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/thetatoken/ukulele/common"
)

// Genesis is the hardcoded genesis checkpoint.
var Genesis = Checkpoint{
	ChainID:    "testchain",
	Epoch:      0,
	Hash:       "a0",
	Validators: []string{"2B30B908BA0D3FCA0706E4F2C8D9D30F5689D541"},
}

// Checkpoint contains metadata of a snapshot of system state.
type Checkpoint struct {
	ChainID    string   `json:"chain_id"`
	Epoch      uint32   `json:"epoch"`
	Hash       string   `json:"hash"`
	Validators []string `json:"validators"`
}

// LoadCheckpoint loads a checkpoint from file system.
func LoadCheckpoint(filePath string) (*Checkpoint, error) {
	r, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	checkpoint := &Checkpoint{}
	dec := json.NewDecoder(r)
	err = dec.Decode(checkpoint)

	// Convert validator IDs to upper case.
	uppers := make([]string, len(checkpoint.Validators))
	for i, v := range checkpoint.Validators {
		uppers[i] = strings.ToUpper(v)
	}
	checkpoint.Validators = uppers

	return checkpoint, err
}

// WriteGenesisCheckpoint writes genesis checkpoint to file system.
func WriteGenesisCheckpoint(filePath string) error {
	raw, err := json.MarshalIndent(Genesis, "", "    ")
	if err != nil {
		return err
	}
	return common.WriteFileAtomic(filePath, raw, 0600)
}
