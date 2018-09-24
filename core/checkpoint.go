package core

// Checkpoint contains metadata of a snapshot of system state.
type Checkpoint struct {
	ChainID    string   `json:"chain_id"`
	Epoch      uint32   `json:"epoch"`
	Hash       string   `json:"hash"`
	Validators []string `json:"validators"`
}
