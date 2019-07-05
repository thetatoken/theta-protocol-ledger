package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto"
)

func TestBlockHash(t *testing.T) {
	assert := assert.New(t)

	eb := &ExtendedBlock{}
	assert.Equal(eb.Hash(), common.Hash{})

	eb = &ExtendedBlock{
		Block: &Block{},
	}
	assert.Equal(eb.Hash(), common.Hash{})

	eb = &ExtendedBlock{
		Block: &Block{
			BlockHeader: &BlockHeader{
				Epoch: 1,
			},
		},
	}
	assert.Equal("0x87a331c1e807476de260f2dc2e4d531dc42500764587605c7574179bc4cbd5bc", eb.Hash().Hex())
}

func TestCreateTestBlock(t *testing.T) {
	assert := assert.New(t)

	b11 := CreateTestBlock("B1", "")
	b12 := CreateTestBlock("b1", "")

	assert.Equal(b11.Hash(), b12.Hash())
}

func TestBlockBasicValidation(t *testing.T) {
	require := require.New(t)
	ResetTestBlocks()

	CreateTestBlock("root", "")
	b1 := CreateTestBlock("B1", "root")
	res := b1.Validate("testchain")
	require.True(res.IsOK())

	res = b1.Validate("anotherchain")
	require.True(res.IsError())
	require.Equal("ChainID mismatch", res.Message)

	oldTS := b1.Timestamp
	b1.Timestamp = nil
	res = b1.Validate("testchain")
	require.True(res.IsError())
	require.Equal("Timestamp is missing", res.Message)
	b1.Timestamp = oldTS

	oldParent := b1.Parent
	b1.Parent = common.Hash{}
	res = b1.Validate("testchain")
	require.True(res.IsError())
	require.Equal("Parent is empty", res.Message)
	b1.Parent = oldParent

	oldProposer := b1.Proposer
	b1.Proposer = common.Address{}
	res = b1.Validate("testchain")
	require.True(res.IsError())
	require.Equal("Proposer is not specified", res.Message)
	b1.Proposer = oldProposer

	oldHCC := b1.HCC
	b1.HCC = CommitCertificate{}
	res = b1.Validate("testchain")
	require.True(res.IsError())
	require.Equal("HCC is empty", res.Message)
	b1.HCC = oldHCC

	oldSig := b1.Signature
	b1.Signature = nil
	res = b1.Validate("testchain")
	require.True(res.IsError())
	require.Equal("Block is not signed", res.Message)
	b1.Signature = oldSig

	oldSig = b1.Signature
	b1.Signature = &crypto.Signature{}
	res = b1.Validate("testchain")
	require.True(res.IsError())
	require.Equal("Block is not signed", res.Message)
	b1.Signature = oldSig

	privKey, _, _ := crypto.GenerateKeyPair()
	sig, _ := privKey.Sign(b1.SignBytes())
	b1.SetSignature(sig)
	res = b1.Validate("testchain")
	require.True(res.IsError())
	require.Equal("Signature verification failed", res.Message)
}
