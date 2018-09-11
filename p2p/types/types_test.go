package types

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/crypto"
	"github.com/thetatoken/ukulele/rlp"
)

func TestNodeInfoRLPEncoding1(t *testing.T) {
	assert := assert.New(t)

	randPrivKey, err := crypto.GenerateKey()
	nodeInfo := CreateNodeInfo(randPrivKey.PublicKey)

	// ------ EncodeToBytes/DecodeBytes ------

	encodedNodeInfoBytes, err := rlp.EncodeToBytes(nodeInfo)
	assert.Nil(err)
	t.Logf("encodedNodeInfoBytes     = %v", string(encodedNodeInfoBytes))

	var decodedNodeInfo NodeInfo
	rlp.DecodeBytes(encodedNodeInfoBytes, &decodedNodeInfo)
	t.Logf("nodeInfo: Address        = %v", nodeInfo.Address)
	t.Logf("decodedNodeINfo: Address = %v", decodedNodeInfo.Address)

	assert.Equal(nodeInfo.Address, decodedNodeInfo.Address)
}

func TestNodeInfoRLPEncoding2(t *testing.T) {
	assert := assert.New(t)

	randPrivKey, err := crypto.GenerateKey()
	nodeInfo := CreateNodeInfo(randPrivKey.PublicKey)

	// ------ Encode/Decode ------

	strBuf := bytes.NewBufferString("")
	err = rlp.Encode(strBuf, nodeInfo)
	assert.Nil(err)

	var decodedNodeInfo NodeInfo
	rlp.Decode(strBuf, &decodedNodeInfo)
	t.Logf("nodeInfo: Address        = %v", nodeInfo.Address)
	t.Logf("decodedNodeINfo: Address = %v", decodedNodeInfo.Address)

	assert.Equal(nodeInfo.Address, decodedNodeInfo.Address)
}
