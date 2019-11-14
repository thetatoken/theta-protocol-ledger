package core

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto"
	"github.com/thetatoken/theta/rlp"
)

func TestBlockEncoding(t *testing.T) {
	require := require.New(t)

	// Serialized block before Gurdian fork.
	oldBlockHash := common.HexToHash("0xf1a7fa371f6a108bb4f2ed33de26ac006f0d8cf6a0ed9dc2c1d9547b6cf43cae")
	v1, err := hex.DecodeString("f90217f902138974657374636861696e0301a035a8f8d3cf9b6da72f72363d53291f9744cab20e420e7e6545235e93a3588e74e2c0a035a8f8d3cf9b6da72f72363d53291f9744cab20e420e7e6545235e93a3588e74a056e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421a056e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421b9010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000b1845dc3ade894c0a4e0c9b349b13b5e882770bfcf20e985691298b841e745098aff2ddbae9aefbb72850f3ff7542dd26bc06252d235f9975a52152a6e257ce7195a31e14b4be0f185b59fd56f5dc3ee444759d16e8ae7964e9370436b00c0")
	require.Nil(err)

	// Should be able to encode/decode blocks before Theta2.0 fork.
	b1 := &Block{}
	err = rlp.DecodeBytes(v1, b1)
	require.Nil(err)

	raw, err := rlp.EncodeToBytes(b1)
	require.Nil(err)
	require.Equal(v1, raw)
	// Block hash should remain the same.
	require.Equal(oldBlockHash, b1.Hash())

	// Should be able to encode/decode blocks before Theta2.0 fork.
	CreateTestBlock("root", "")
	b2 := CreateTestBlock("b2", "root")
	b2.AddTxs([]common.Bytes{common.Hex2Bytes("aaa")})
	b2raw1, _ := rlp.EncodeToBytes(b2)
	tmp := &Block{}
	err = rlp.DecodeBytes(b2raw1, tmp)
	require.Nil(err)
	b2raw2, _ := rlp.EncodeToBytes(tmp)
	require.Equal(b2raw1, b2raw2)

	// Should be able to encode/decode blocks after Theta2.0 fork.
	b2.Height = common.HeightEnableTheta2
	b2raw1, _ = rlp.EncodeToBytes(b2)
	err = rlp.DecodeBytes(b2raw1, tmp)
	require.Nil(err)
	b2raw2, _ = rlp.EncodeToBytes(tmp)
	require.Equal(b2raw1, b2raw2)

	// Decode with guardian votes.
	b2.GuardianVotes = NewAggregateVotes(b2.Hash(), NewGuardianCandidatePool())
	b2raw1, _ = rlp.EncodeToBytes(b2)
	err = rlp.DecodeBytes(b2raw1, tmp)
	require.Nil(err)
	b2raw2, _ = rlp.EncodeToBytes(tmp)
	require.Equal(b2raw1, b2raw2)
	require.Equal(tmp.GuardianVotes.Block, b2.GuardianVotes.Block)

	// Test ExtendedBlock encoding/decoding
	eb := &ExtendedBlock{}
	eb.Block = b2
	eb.Children = []common.Hash{eb.Hash()}
	eb.Status = BlockStatusCommitted
	eb.HasValidatorUpdate = true
	ebraw1, _ := rlp.EncodeToBytes(eb)

	tmp2 := &ExtendedBlock{}
	err = rlp.DecodeBytes(ebraw1, tmp2)
	require.Nil(err)
	ebraw2, _ := rlp.EncodeToBytes(tmp2)
	require.Equal(ebraw1, ebraw2)

	_, err = rlp.EncodeToBytes(tmp2)
	require.Nil(err)
}

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
