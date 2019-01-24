package types

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/crypto"
	"github.com/thetatoken/theta/rlp"
)

func TestNodeInfoRLPEncoding1(t *testing.T) {
	assert := assert.New(t)

	_, randPubKey, err := crypto.GenerateKeyPair()
	nodeInfo := CreateNodeInfo(randPubKey, 1234)

	// ------ EncodeToBytes/DecodeBytes ------

	encodedNodeInfoBytes, err := rlp.EncodeToBytes(nodeInfo)
	assert.Nil(err)
	t.Logf("encodedNodeInfoBytes     = %v", hex.EncodeToString(encodedNodeInfoBytes))

	var decodedNodeInfo NodeInfo
	rlp.DecodeBytes(encodedNodeInfoBytes, &decodedNodeInfo)
	decodedNodeInfo.PubKey, err = crypto.PublicKeyFromBytes(decodedNodeInfo.PubKeyBytes)
	assert.Nil(err)

	t.Logf("nodeInfo: Address        = %v", nodeInfo.PubKey.Address().Hex())
	t.Logf("decodedNodeINfo: Address = %v", decodedNodeInfo.PubKey.Address().Hex())

	assert.Equal(nodeInfo.PubKey.Address(), decodedNodeInfo.PubKey.Address())
}

func TestNodeInfoRLPEncoding2(t *testing.T) {
	assert := assert.New(t)

	_, randPubKey, err := crypto.GenerateKeyPair()
	nodeInfo := CreateNodeInfo(randPubKey, 1234)

	// ------ Encode/Decode ------

	strBuf := bytes.NewBufferString("")
	err = rlp.Encode(strBuf, nodeInfo)
	assert.Nil(err)

	var decodedNodeInfo NodeInfo
	rlp.Decode(strBuf, &decodedNodeInfo)
	decodedNodeInfo.PubKey, err = crypto.PublicKeyFromBytes(decodedNodeInfo.PubKeyBytes)
	assert.Nil(err)

	t.Logf("nodeInfo: Address        = %v", nodeInfo.PubKey.Address().Hex())
	t.Logf("decodedNodeINfo: Address = %v", decodedNodeInfo.PubKey.Address().Hex())

	assert.Equal(nodeInfo.PubKey.Address(), decodedNodeInfo.PubKey.Address())
}
