package core

import "fmt"

type BackupBlock struct {
	Block *ExtendedBlock
	Votes *VoteSet `rlp:"nil"`
}

func (b *BackupBlock) String() string {
	return fmt.Sprintf("BackupBlock{Block: %v, Votes: %v", b.Block.String(), b.Votes.String())
}
