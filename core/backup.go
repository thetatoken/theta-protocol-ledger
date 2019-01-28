package core

type BackupBlock struct {
	Block *Block
	Votes *VoteSet `rlp:"nil"`
}
