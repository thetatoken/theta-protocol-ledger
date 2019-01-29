package core

import "fmt"

type BackupBlock struct {
	Block *ExtendedBlock
	Votes *VoteSet `rlp:"nil"`
	Next  *BackupBlock
}

func (b *BackupBlock) String() string {
	return fmt.Sprintf("BackupBlock{Block: %v, Votes: %v, Next: %v", b.Block.String(), b.Votes.String(), b.Next)
}
