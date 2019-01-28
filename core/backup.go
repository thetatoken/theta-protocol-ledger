package core

type BackupBlock struct {
	Block *ExtendedBlock
	Votes *VoteSet `rlp:"nil"`
}
