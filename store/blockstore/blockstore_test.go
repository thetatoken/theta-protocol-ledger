package blockstore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/blockchain"
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

	parent := &blockchain.Block{}
	parent.ChainID = "testparent"
	parent.Epoch = 0
	parent.Hash = blockchain.ParseHex("a0")
	extendedParent := blockchain.ExtendedBlock{Block: parent, Height: 6, Children: nil, Parent: nil, CommitCertificate: nil}

	child1 := &blockchain.Block{}
	child1.ChainID = "testchild1"
	child1.Epoch = 2
	child1.Hash = blockchain.ParseHex("c1")
	extendedChild1 := blockchain.ExtendedBlock{Block: child1, Height: 8, Children: nil, Parent: &extendedParent, CommitCertificate: nil}

	child2 := &blockchain.Block{}
	child2.ChainID = "testchild2"
	child2.Epoch = 3
	child2.Hash = blockchain.ParseHex("c2")
	extendedChild2 := blockchain.ExtendedBlock{Block: child2, Height: 8, Children: nil, Parent: &extendedParent, CommitCertificate: nil}

	block := &blockchain.Block{}
	block.ChainID = "testblock"
	block.Epoch = 1
	block.Hash = blockchain.ParseHex("b1")
	extendedBlock := blockchain.ExtendedBlock{Block: block, Height: 7, Children: []*blockchain.ExtendedBlock{&extendedChild1, &extendedChild2}, Parent: &extendedParent, CommitCertificate: nil}

	key2 := blockchain.ParseHex("def321")
	err = blockStore.Put(key2, extendedBlock)
	assert.Nil(err)

	var val blockchain.ExtendedBlock
	err = blockStore.Get(key2, &val)
	assert.Nil(err)
	assert.Equal(extendedBlock.Height, val.Height)
	assert.Equal(extendedBlock.Block.ChainID, val.Block.ChainID)
	assert.Equal(extendedBlock.Block.Epoch, val.Block.Epoch)
	assert.Equal(extendedBlock.Block.Hash, val.Block.Hash)
	assert.Equal(extendedBlock.Parent.ChainID, val.Parent.ChainID)
	assert.Equal(extendedBlock.Parent.Epoch, val.Parent.Epoch)
	assert.Equal(extendedBlock.Parent.Hash, val.Parent.Hash)
	assert.Equal(2, len(val.Children))
	assert.Equal(extendedBlock.Children[0].ChainID, val.Children[0].ChainID)
	assert.Equal(extendedBlock.Children[0].Epoch, val.Children[0].Epoch)
	assert.Equal(extendedBlock.Children[0].Hash, val.Children[0].Hash)
	assert.Equal(extendedBlock.Children[1].ChainID, val.Children[1].ChainID)
	assert.Equal(extendedBlock.Children[1].Epoch, val.Children[1].Epoch)
	assert.Equal(extendedBlock.Children[1].Hash, val.Children[1].Hash)

	err = blockStore.Delete(key2)
	assert.Nil(err)

	var res2 blockchain.ExtendedBlock
	err = blockStore.Get(key2, &res2)
	assert.NotNil(err)
	assert.Equal(store.ErrKeyNotFound, err)
}
