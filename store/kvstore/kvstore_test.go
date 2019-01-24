// +build cluster_deployment

package kvstore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/store"
	"github.com/thetatoken/theta/store/database/backend"
)

type NodeHeader struct {
	ChainID    string
	Epoch      uint64
	Hash       common.Bytes
	ParentHash common.Bytes
}

type Record struct {
}

type Node struct {
	NodeHeader
	Records []Record
}

type Choice struct {
	ID string
}

type ChoiceSet struct {
	Choices []Choice
}

type Certificate struct {
	Choices  *ChoiceSet `rlp:"nil"`
	NodeHash common.Bytes
}

type ExtendedNode struct {
	*Node
	Height      uint64
	Children    []common.Bytes
	Parent      common.Bytes
	Certificate *Certificate `rlp:"nil"`
}

func TestKVStore(t *testing.T) {
	assert := assert.New(t)

	db, err := backend.NewMgoDatabase()
	assert.Nil(err)
	kvstore := NewKVStore(db)

	key := []byte("abc123")

	err = kvstore.Put(key, "hello!")
	assert.Nil(err)

	var str string
	err = kvstore.Get(key, &str)
	assert.Nil(err)
	assert.Equal("hello!", str)

	err = kvstore.Delete(key)
	assert.Nil(err)

	err = kvstore.Get(key, &str)
	assert.NotNil(err)
	assert.Equal(store.ErrKeyNotFound, err)

	nodeHash := common.Bytes("b1")

	parent := &Node{}
	parent.ChainID = "testparent"
	parent.Epoch = 0
	parent.Hash = []byte("a0")
	extendedParent := ExtendedNode{Node: parent, Height: 6, Children: []common.Bytes{nodeHash}, Parent: nil, Certificate: nil}

	child1 := &Node{}
	child1.ChainID = "testchild1"
	child1.Epoch = 2
	child1.Hash = []byte("c1")
	extendedChild1 := ExtendedNode{Node: child1, Height: 8, Children: nil, Parent: extendedParent.Hash, Certificate: nil}

	child2 := &Node{}
	child2.ChainID = "testchild2"
	child2.Epoch = 3
	child2.Hash = []byte("c2")
	extendedChild2 := ExtendedNode{Node: child2, Height: 8, Children: nil, Parent: extendedParent.Hash, Certificate: nil}

	node := &Node{}
	node.ChainID = "testblock"
	node.Epoch = 1
	node.Hash = nodeHash
	nodeHash = []byte("zz")
	choice1 := Choice{"aa"}
	choice2 := Choice{"bb"}
	choices := ChoiceSet{[]Choice{choice1, choice2}}
	cert := Certificate{NodeHash: nodeHash, Choices: &choices}
	extendedNode := ExtendedNode{Node: node, Height: 7, Children: []common.Bytes{extendedChild1.Hash, extendedChild2.Hash}, Parent: extendedParent.Hash, Certificate: &cert}

	err = kvstore.Put(extendedParent.Hash, extendedParent)
	assert.Nil(err)

	err = kvstore.Put(extendedNode.Hash, extendedNode)
	assert.Nil(err)

	err = kvstore.Put(extendedChild1.Hash, extendedChild1)
	assert.Nil(err)

	err = kvstore.Put(extendedChild2.Hash, extendedChild2)
	assert.Nil(err)

	var nodeVal ExtendedNode
	err = kvstore.Get(extendedNode.Hash, &nodeVal)
	assert.Nil(err)
	assert.Equal(extendedNode.Height, nodeVal.Height)
	assert.Equal(extendedNode.Node.Hash, nodeVal.Node.Hash)
	assert.Equal(extendedNode.Node.ChainID, nodeVal.Node.ChainID)
	assert.Equal(extendedNode.Node.Epoch, nodeVal.Node.Epoch)
	assert.Equal(nodeHash, nodeVal.Certificate.NodeHash)

	assert.Equal(choice1.ID, (*(nodeVal.Certificate.Choices)).Choices[0].ID)
	assert.Equal(choice2.ID, (*(nodeVal.Certificate.Choices)).Choices[1].ID)

	assert.Equal(2, len(nodeVal.Children))
	assert.Equal(extendedNode.Children[0], nodeVal.Children[0])
	assert.Equal(extendedNode.Children[1], nodeVal.Children[1])

	var parentVal ExtendedNode
	err = kvstore.Get(nodeVal.Parent, &parentVal)
	assert.Nil(err)
	assert.Equal(extendedParent.Height, parentVal.Height)
	assert.Equal(extendedParent.Node.Hash, parentVal.Node.Hash)
	assert.Equal(extendedParent.Node.ChainID, parentVal.Node.ChainID)
	assert.Equal(extendedParent.Node.Epoch, parentVal.Node.Epoch)
	assert.Empty(parentVal.Parent)
	assert.Equal(1, len(parentVal.Children))
	assert.Equal(parentVal.Children[0], nodeVal.Hash)

	err = kvstore.Delete(nodeVal.Parent)
	err = kvstore.Delete(nodeVal.Children[0])
	err = kvstore.Delete(nodeVal.Children[1])
	err = kvstore.Delete(nodeVal.Hash)
	assert.Nil(err)

	var res ExtendedNode
	err = kvstore.Get(extendedNode.Hash, &res)
	assert.NotNil(err)
	assert.Equal(store.ErrKeyNotFound, err)
}
