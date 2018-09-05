package blockstore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/store"
	"github.com/thetatoken/ukulele/store/database/backend"
)

type BlockHeader struct {
	ChainID    string
	Epoch      uint32
	Hash       common.Bytes
	ParentHash common.Bytes
}

type Tx struct {
}

type Block struct {
	BlockHeader
	Txs []Tx
}

type Vote struct {
	Block *BlockHeader
	ID    string
}

type VoteSet struct {
	votes []Vote
}

type CommitCertificate struct {
	Votes     *VoteSet `rlp:"nil"`
	BlockHash common.Bytes
}

type ExtendedBlock struct {
	*Block
	Height            uint32
	Children          []common.Bytes
	Parent            common.Bytes
	CommitCertificate *CommitCertificate `rlp:"nil"`
}

func TestBlockStore(t *testing.T) {
	assert := assert.New(t)

	db, err := backend.NewMgoDatabase()
	assert.Nil(err)
	blockStore := NewBlockStore(db)

	key := []byte("abc123")

	err = blockStore.Put(key, "hello!")
	assert.Nil(err)

	var str string
	err = blockStore.Get(key, &str)
	assert.Nil(err)
	assert.Equal("hello!", str)

	err = blockStore.Delete(key)
	assert.Nil(err)

	err = blockStore.Get(key, &str)
	assert.NotNil(err)
	assert.Equal(store.ErrKeyNotFound, err)

	blockHash := []byte("b1")

	parent := &Block{}
	parent.ChainID = "testparent"
	parent.Epoch = 0
	parent.Hash = []byte("a0")
	extendedParent := ExtendedBlock{Block: parent, Height: 6, Children: []common.Bytes{blockHash}, Parent: nil, CommitCertificate: nil}

	child1 := &Block{}
	child1.ChainID = "testchild1"
	child1.Epoch = 2
	child1.Hash = []byte("c1")
	extendedChild1 := ExtendedBlock{Block: child1, Height: 8, Children: nil, Parent: extendedParent.Hash, CommitCertificate: nil}

	child2 := &Block{}
	child2.ChainID = "testchild2"
	child2.Epoch = 3
	child2.Hash = []byte("c2")
	extendedChild2 := ExtendedBlock{Block: child2, Height: 8, Children: nil, Parent: extendedParent.Hash, CommitCertificate: nil}

	block := &Block{}
	block.ChainID = "testblock"
	block.Epoch = 1
	block.Hash = blockHash
	extendedBlock := ExtendedBlock{Block: block, Height: 7, Children: []common.Bytes{extendedChild1.Hash, extendedChild2.Hash}, Parent: extendedParent.Hash, CommitCertificate: nil}

	err = blockStore.Put(extendedParent.Hash, extendedParent)
	assert.Nil(err)

	err = blockStore.Put(extendedBlock.Hash, extendedBlock)
	assert.Nil(err)

	err = blockStore.Put(extendedChild1.Hash, extendedChild1)
	assert.Nil(err)

	err = blockStore.Put(extendedChild2.Hash, extendedChild2)
	assert.Nil(err)

	var blockVal ExtendedBlock
	err = blockStore.Get(extendedBlock.Hash, &blockVal)
	assert.Nil(err)
	assert.Equal(extendedBlock.Height, blockVal.Height)
	assert.Equal(extendedBlock.Block.Hash, blockVal.Block.Hash)
	assert.Equal(extendedBlock.Block.ChainID, blockVal.Block.ChainID)
	assert.Equal(extendedBlock.Block.Epoch, blockVal.Block.Epoch)
	assert.Equal(2, len(blockVal.Children))
	assert.Equal(extendedBlock.Children[0], blockVal.Children[0])
	assert.Equal(extendedBlock.Children[1], blockVal.Children[1])

	var parentVal ExtendedBlock
	err = blockStore.Get(blockVal.Parent, &parentVal)
	assert.Nil(err)
	assert.Equal(extendedParent.Height, parentVal.Height)
	assert.Equal(extendedParent.Block.Hash, parentVal.Block.Hash)
	assert.Equal(extendedParent.Block.ChainID, parentVal.Block.ChainID)
	assert.Equal(extendedParent.Block.Epoch, parentVal.Block.Epoch)
	assert.Empty(parentVal.Parent)
	assert.Equal(1, len(parentVal.Children))
	assert.Equal(parentVal.Children[0], blockVal.Hash)

	err = blockStore.Delete(blockVal.Parent)
	err = blockStore.Delete(blockVal.Children[0])
	err = blockStore.Delete(blockVal.Children[1])
	err = blockStore.Delete(blockVal.Hash)
	assert.Nil(err)

	var res ExtendedBlock
	err = blockStore.Get(extendedBlock.Hash, &res)
	assert.NotNil(err)
	assert.Equal(store.ErrKeyNotFound, err)
}
