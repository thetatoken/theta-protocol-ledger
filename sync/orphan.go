package sync

import (
	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/common"
)

type OrphanBlockPool struct{}

func NewOrphanBlockPool() *OrphanBlockPool {
	return nil
}

func (obp *OrphanBlockPool) Add(block *blockchain.Block) {

}

func (obp *OrphanBlockPool) TryGetNextBlock(hash common.Bytes) *blockchain.Block {
	return nil
}

type OrphanCCPool struct{}

func NewOrphanCCPool() *OrphanCCPool {
	return nil
}

func (ocp *OrphanCCPool) Add(cc *blockchain.CommitCertificate) {

}

func (ocp *OrphanCCPool) TryGetCCByBlock(hash common.Bytes) *blockchain.CommitCertificate {
	return nil
}
