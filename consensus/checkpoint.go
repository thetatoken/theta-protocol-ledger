package consensus

import (
	"encoding/json"
	"os"
)

// Checkpoint contains metadata of a snapshot of system state.
type Checkpoint struct {
	ChainID       string
	StartingEpoch uint32
	StartingHash  string
	Validators    []string
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
	return checkpoint, err
}
