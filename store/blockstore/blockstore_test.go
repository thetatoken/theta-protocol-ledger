package blockstore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/store"
)

func TestBlockStore(t *testing.T) {
	assert := assert.New(t)

	blockStore := NewBlockStore()

	key := blockchain.ParseHex("abc123")

	err := blockStore.Put(key, "hello!")
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

	blockHash := blockchain.ParseHex("b1")

	parent := &blockchain.Block{}
	parent.ChainID = "testparent"
	parent.Epoch = 0
	parent.Hash = blockchain.ParseHex("a0")
	extendedParent := blockchain.ExtendedBlock{Block: parent, Height: 6, Children: []common.Bytes{blockHash}, Parent: nil, CommitCertificate: nil}

	child1 := &blockchain.Block{}
	child1.ChainID = "testchild1"
	child1.Epoch = 2
	child1.Hash = blockchain.ParseHex("c1")
	extendedChild1 := blockchain.ExtendedBlock{Block: child1, Height: 8, Children: nil, Parent: extendedParent.Hash, CommitCertificate: nil}

	child2 := &blockchain.Block{}
	child2.ChainID = "testchild2"
	child2.Epoch = 3
	child2.Hash = blockchain.ParseHex("c2")
	extendedChild2 := blockchain.ExtendedBlock{Block: child2, Height: 8, Children: nil, Parent: extendedParent.Hash, CommitCertificate: nil}

	block := &blockchain.Block{}
	block.ChainID = "testblock"
	block.Epoch = 1
	block.Hash = blockHash
	extendedBlock := blockchain.ExtendedBlock{Block: block, Height: 7, Children: []common.Bytes{extendedChild1.Hash, extendedChild2.Hash}, Parent: extendedParent.Hash, CommitCertificate: nil}

	err = blockStore.Put(extendedParent.Hash, extendedParent)
	assert.Nil(err)

	err = blockStore.Put(extendedBlock.Hash, extendedBlock)
	assert.Nil(err)

	err = blockStore.Put(extendedChild1.Hash, extendedChild1)
	assert.Nil(err)

	err = blockStore.Put(extendedChild2.Hash, extendedChild2)
	assert.Nil(err)

	var blockVal blockchain.ExtendedBlock
	err = blockStore.Get(extendedBlock.Hash, &blockVal)
	assert.Nil(err)
	assert.Equal(extendedBlock.Height, blockVal.Height)
	assert.Equal(extendedBlock.Block.Hash, blockVal.Block.Hash)
	assert.Equal(extendedBlock.Block.ChainID, blockVal.Block.ChainID)
	assert.Equal(extendedBlock.Block.Epoch, blockVal.Block.Epoch)
	assert.Equal(2, len(blockVal.Children))
	assert.Equal(extendedBlock.Children[0], blockVal.Children[0])
	assert.Equal(extendedBlock.Children[1], blockVal.Children[1])

	var parentVal blockchain.ExtendedBlock
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

	var res blockchain.ExtendedBlock
	err = blockStore.Get(extendedBlock.Hash, &res)
	assert.NotNil(err)
	assert.Equal(store.ErrKeyNotFound, err)
}
