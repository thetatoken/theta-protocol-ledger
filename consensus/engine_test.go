package consensus

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/crypto"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/store/database/backend"
	"github.com/thetatoken/ukulele/store/kvstore"
)

type MockValidatorManager struct {
	PrivKey *crypto.PrivateKey
}

func (m MockValidatorManager) GetProposer(_ common.Hash, _ uint64) core.Validator {
	return core.NewValidator(m.PrivKey.PublicKey().Address().Hex(), types.Zero)
}

func (m MockValidatorManager) GetValidatorSet(_ common.Hash) *core.ValidatorSet {
	v := core.NewValidatorSet()
	v.AddValidator(m.GetProposer(common.Hash{}, 0))
	return v
}

func (m MockValidatorManager) SetConsensusEngine(consensus core.ConsensusEngine) {}

func TestSingleBlockValidation(t *testing.T) {
	require := require.New(t)

	privKey, _, _ := crypto.GenerateKeyPair()
	validatorManager := MockValidatorManager{PrivKey: privKey}

	store := kvstore.NewKVStore(backend.NewMemDatabase())
	root := core.CreateTestBlock("a0", "")
	root.ChainID = "testchain"
	root.Epoch = 0
	chain := blockchain.NewChain("testchain", store, root)

	ce := NewConsensusEngine(nil, store, chain, nil, validatorManager)

	// Valid block
	b1 := core.NewBlock()
	b1.ChainID = chain.ChainID
	b1.Height = chain.Root().Height + 1
	b1.Epoch = 1
	b1.Parent = chain.Root().Hash()
	b1.HCC = b1.Parent
	b1.Proposer = privKey.PublicKey().Address()
	b1.Timestamp = big.NewInt(time.Now().Unix())
	b1.Signature, _ = privKey.Sign(b1.SignBytes())
	chain.AddBlock(b1)

	require.True(ce.validateBlock(b1, chain.Root()))

	// Invalid blocks.
	invalidBlock := core.NewBlock()
	invalidBlock.ChainID = chain.ChainID
	invalidBlock.Epoch = 2
	invalidBlock.Parent = chain.Root().Hash()
	invalidBlock.HCC = invalidBlock.Parent
	invalidBlock.Timestamp = big.NewInt(time.Now().Unix())
	invalidBlock.Proposer = privKey.PublicKey().Address()
	invalidBlock.Signature, _ = privKey.Sign(invalidBlock.SignBytes())
	_, err := chain.AddBlock(invalidBlock)
	require.Nil(err)
	require.False(ce.validateBlock(invalidBlock, chain.Root()), "Missing height")

	invalidBlock = core.NewBlock()
	invalidBlock.ChainID = chain.ChainID
	invalidBlock.Height = 1
	invalidBlock.Parent = chain.Root().Hash()
	invalidBlock.HCC = invalidBlock.Parent
	invalidBlock.Timestamp = big.NewInt(time.Now().Unix())
	invalidBlock.Proposer = privKey.PublicKey().Address()
	invalidBlock.Signature, _ = privKey.Sign(invalidBlock.SignBytes())
	_, err = chain.AddBlock(invalidBlock)
	require.Nil(err)
	require.False(ce.validateBlock(invalidBlock, chain.Root()), "Missing epoch")

	invalidBlock = core.NewBlock()
	invalidBlock.ChainID = chain.ChainID
	invalidBlock.Height = 1
	invalidBlock.Epoch = 3
	invalidBlock.Parent = common.Hash{}
	invalidBlock.HCC = invalidBlock.Parent
	invalidBlock.Timestamp = big.NewInt(time.Now().Unix())
	invalidBlock.Proposer = privKey.PublicKey().Address()
	invalidBlock.Signature, _ = privKey.Sign(invalidBlock.SignBytes())
	_, err = chain.AddBlock(invalidBlock)
	require.Nil(err)
	require.False(ce.validateBlock(invalidBlock, chain.Root()), "Missing parent")

	invalidBlock = core.NewBlock()
	invalidBlock.ChainID = chain.ChainID
	invalidBlock.Height = 1
	invalidBlock.Epoch = 4
	invalidBlock.Parent = chain.Root().Hash()
	invalidBlock.HCC = common.Hash{}
	invalidBlock.Timestamp = big.NewInt(time.Now().Unix())
	invalidBlock.Proposer = privKey.PublicKey().Address()
	invalidBlock.Signature, _ = privKey.Sign(invalidBlock.SignBytes())
	_, err = chain.AddBlock(invalidBlock)
	require.Nil(err)
	require.False(ce.validateBlock(invalidBlock, chain.Root()), "Missing HCC")

	invalidBlock = core.NewBlock()
	invalidBlock.ChainID = chain.ChainID
	invalidBlock.Height = 1
	invalidBlock.Epoch = 5
	invalidBlock.Parent = chain.Root().Hash()
	invalidBlock.HCC = invalidBlock.Parent
	invalidBlock.Timestamp = big.NewInt(time.Now().Unix())
	invalidBlock.Proposer = common.Address{}
	invalidBlock.Signature, _ = privKey.Sign(invalidBlock.SignBytes())
	_, err = chain.AddBlock(invalidBlock)
	require.Nil(err)
	require.False(ce.validateBlock(invalidBlock, chain.Root()), "Missing Proposer")

	invalidBlock = core.NewBlock()
	invalidBlock.ChainID = chain.ChainID
	invalidBlock.Height = 1
	invalidBlock.Epoch = 6
	invalidBlock.Parent = chain.Root().Hash()
	invalidBlock.HCC = invalidBlock.Parent
	invalidBlock.Proposer = privKey.PublicKey().Address()
	invalidBlock.Timestamp = big.NewInt(time.Now().Unix())

	privKey2, _, _ := crypto.GenerateKeyPair()
	invalidBlock.Signature, _ = privKey2.Sign(invalidBlock.SignBytes())
	_, err = chain.AddBlock(invalidBlock)
	require.Nil(err)
	require.False(ce.validateBlock(invalidBlock, chain.Root()), "Invalid signature")

	invalidBlock = core.NewBlock()
	invalidBlock.ChainID = chain.ChainID
	invalidBlock.Height = 1
	invalidBlock.Epoch = 6
	invalidBlock.Parent = chain.Root().Hash()
	invalidBlock.HCC = invalidBlock.Parent
	invalidBlock.Signature, _ = privKey.Sign(invalidBlock.SignBytes())
	invalidBlock.Proposer = privKey.PublicKey().Address()
	_, err = chain.AddBlock(invalidBlock)
	require.Nil(err)
	require.False(ce.validateBlock(invalidBlock, chain.Root()), "Missing timestamp")
}

func TestValidParent(t *testing.T) {
	require := require.New(t)

	privKey, _, _ := crypto.GenerateKeyPair()
	validatorManager := MockValidatorManager{PrivKey: privKey}

	store := kvstore.NewKVStore(backend.NewMemDatabase())
	root := core.CreateTestBlock("a0", "")
	root.ChainID = "testchain"
	root.Epoch = 0
	chain := blockchain.NewChain("testchain", store, root)

	ce := NewConsensusEngine(nil, store, chain, nil, validatorManager)

	b1 := core.NewBlock()
	b1.ChainID = chain.ChainID
	b1.Height = chain.Root().Height + 1
	b1.Epoch = 1
	b1.Parent = chain.Root().Hash()
	b1.HCC = b1.Parent
	b1.Proposer = privKey.PublicKey().Address()
	b1.Timestamp = big.NewInt(time.Now().Unix())
	b1.Signature, _ = privKey.Sign(b1.SignBytes())
	eb1, err := chain.AddBlock(b1)
	require.Nil(err)

	b2 := core.NewBlock()
	b2.ChainID = chain.ChainID
	b2.Height = 2
	b2.Epoch = 2
	b2.Parent = b1.Hash()
	b2.HCC = b2.Parent
	b2.Proposer = privKey.PublicKey().Address()
	b2.Timestamp = big.NewInt(time.Now().Unix())
	b2.Signature, _ = privKey.Sign(b2.SignBytes())
	eb2, err := chain.AddBlock(b2)
	require.Nil(err)

	require.False(ce.validateBlock(b2, eb1), "Parent block is invalid")

	// HCC: b1 <= b2
	eb1 = chain.MarkBlockValid(eb1.Hash())
	require.True(ce.validateBlock(b2, eb1), "Parent block is valid")

	// Validator updating block's child
	b3 := core.NewBlock()
	b3.ChainID = chain.ChainID
	b3.Height = 3
	b3.Epoch = 3
	b3.Parent = b2.Hash()
	// b3's HCC is linked to b1
	b3.HCC = b1.Hash()
	b3.Proposer = privKey.PublicKey().Address()
	b3.Timestamp = big.NewInt(time.Now().Unix())
	b3.Signature, _ = privKey.Sign(b3.SignBytes())
	_, err = chain.AddBlock(b3)
	require.Nil(err)
	eb2 = chain.MarkBlockValid(eb2.Hash())
	require.True(ce.validateBlock(b3, eb2), "HCC is valid")
}

func TestChildBlockOfValidatorChange(t *testing.T) {
	require := require.New(t)

	privKey, _, _ := crypto.GenerateKeyPair()
	validatorManager := MockValidatorManager{PrivKey: privKey}

	store := kvstore.NewKVStore(backend.NewMemDatabase())
	root := core.CreateTestBlock("a0", "")
	root.ChainID = "testchain"
	root.Epoch = 0
	chain := blockchain.NewChain("testchain", store, root)

	ce := NewConsensusEngine(nil, store, chain, nil, validatorManager)

	b1 := core.NewBlock()
	b1.ChainID = chain.ChainID
	b1.Height = chain.Root().Height + 1
	b1.Epoch = 1
	b1.Parent = chain.Root().Hash()
	b1.HCC = b1.Parent
	b1.Proposer = privKey.PublicKey().Address()
	b1.Timestamp = big.NewInt(time.Now().Unix())
	b1.Signature, _ = privKey.Sign(b1.SignBytes())
	eb1, err := chain.AddBlock(b1)
	require.Nil(err)

	b2 := core.NewBlock()
	b2.ChainID = chain.ChainID
	b2.Height = 2
	b2.Epoch = 2
	b2.Parent = b1.Hash()
	b2.HCC = b2.Parent
	b2.Proposer = privKey.PublicKey().Address()
	b2.Timestamp = big.NewInt(time.Now().Unix())
	b2.Signature, _ = privKey.Sign(b2.SignBytes())
	eb2, err := chain.AddBlock(b2)
	require.Nil(err)

	eb1 = chain.MarkBlockValid(eb1.Hash())
	eb2 = chain.MarkBlockValid(eb2.Hash())

	// Validator updating block's child
	b3 := core.NewBlock()
	b3.ChainID = chain.ChainID
	b3.Height = 3
	b3.Epoch = 3
	b3.Parent = b2.Hash()
	// b3's HCC is linked to b1
	b3.HCC = b1.Hash()
	b3.Proposer = privKey.PublicKey().Address()
	b3.Timestamp = big.NewInt(time.Now().Unix())
	b3.Signature, _ = privKey.Sign(b3.SignBytes())
	_, err = chain.AddBlock(b3)
	require.Nil(err)
	require.True(ce.validateBlock(b3, eb2), "HCC is valid")

	// b2 is now marked to have validator changes.
	eb2 = chain.MarkBlockHasValidatorUpdate(eb2.Hash())
	require.False(ce.validateBlock(b3, eb2), "Block with validator update need to be its child's HCC")

	// Validator updating block's child.
	b3 = core.NewBlock()
	b3.ChainID = chain.ChainID
	b3.Height = 3
	b3.Epoch = 4
	b3.Parent = b2.Hash()
	b3.HCC = b2.Hash()
	b3.Proposer = privKey.PublicKey().Address()
	b3.Timestamp = big.NewInt(time.Now().Unix())
	b3.Signature, _ = privKey.Sign(b3.SignBytes())
	_, err = chain.AddBlock(b3)
	require.Nil(err)
	require.True(ce.validateBlock(b3, eb2), "HCC is valid")
}

func TestGrandChildBlockOfValidatorChange(t *testing.T) {
	require := require.New(t)

	privKey, _, _ := crypto.GenerateKeyPair()
	validatorManager := MockValidatorManager{PrivKey: privKey}

	store := kvstore.NewKVStore(backend.NewMemDatabase())
	root := core.CreateTestBlock("a0", "")
	root.ChainID = "testchain"
	root.Epoch = 0
	chain := blockchain.NewChain("testchain", store, root)

	ce := NewConsensusEngine(nil, store, chain, nil, validatorManager)

	b1 := core.NewBlock()
	b1.ChainID = chain.ChainID
	b1.Height = chain.Root().Height + 1
	b1.Epoch = 1
	b1.Parent = chain.Root().Hash()
	b1.HCC = b1.Parent
	b1.Proposer = privKey.PublicKey().Address()
	b1.Timestamp = big.NewInt(time.Now().Unix())
	b1.Signature, _ = privKey.Sign(b1.SignBytes())
	eb1, err := chain.AddBlock(b1)
	require.Nil(err)
	eb1 = chain.MarkBlockValid(eb1.Hash())

	b2 := core.NewBlock()
	b2.ChainID = chain.ChainID
	b2.Height = 2
	b2.Epoch = 2
	b2.Parent = b1.Hash()
	b2.HCC = b2.Parent
	b2.Proposer = privKey.PublicKey().Address()
	b2.Timestamp = big.NewInt(time.Now().Unix())
	b2.Signature, _ = privKey.Sign(b2.SignBytes())
	eb2, err := chain.AddBlock(b2)
	require.Nil(err)
	eb2 = chain.MarkBlockValid(eb2.Hash())
	eb2 = chain.MarkBlockHasValidatorUpdate(eb2.Hash())

	// Validator updating block's child
	b3 := core.NewBlock()
	b3.ChainID = chain.ChainID
	b3.Height = 3
	b3.Epoch = 3
	b3.Parent = b2.Hash()
	b3.HCC = b2.Hash()
	b3.Proposer = privKey.PublicKey().Address()
	b3.Timestamp = big.NewInt(time.Now().Unix())
	b3.Signature, _ = privKey.Sign(b3.SignBytes())
	_, err = chain.AddBlock(b3)
	require.Nil(err)
	eb3 := chain.MarkBlockValid(b3.Hash())

	// Valid grand child.
	b4 := core.NewBlock()
	b4.ChainID = chain.ChainID
	b4.Height = 4
	b4.Epoch = 5
	b4.Parent = b3.Hash()
	b4.HCC = b3.Hash()
	b4.Proposer = privKey.PublicKey().Address()
	b4.Timestamp = big.NewInt(time.Now().Unix())
	b4.Signature, _ = privKey.Sign(b4.SignBytes())
	_, err = chain.AddBlock(b4)
	require.Nil(err)
	require.True(ce.validateBlock(b4, eb3), "HCC is valid")

	// Invalid grand child: HCC link to b2.
	b4 = core.NewBlock()
	b4.ChainID = chain.ChainID
	b4.Height = 4
	b4.Epoch = 6
	b4.Parent = b3.Hash()
	b4.HCC = b2.Hash()
	b4.Proposer = privKey.PublicKey().Address()
	b4.Timestamp = big.NewInt(time.Now().Unix())
	b4.Signature, _ = privKey.Sign(b4.SignBytes())
	_, err = chain.AddBlock(b4)
	require.Nil(err)
	require.False(ce.validateBlock(b4, eb3), "HCC is valid")

	// Invalid grand child: HCC link to b1.
	b4 = core.NewBlock()
	b4.ChainID = chain.ChainID
	b4.Height = 4
	b4.Epoch = 7
	b4.Parent = b3.Hash()
	b4.HCC = b1.Hash()
	b4.Proposer = privKey.PublicKey().Address()
	b4.Timestamp = big.NewInt(time.Now().Unix())
	b4.Signature, _ = privKey.Sign(b4.SignBytes())
	_, err = chain.AddBlock(b4)
	require.Nil(err)
	require.False(ce.validateBlock(b4, eb3), "HCC is valid")
}

func TestGrandGrandChildBlockOfValidatorChange(t *testing.T) {
	require := require.New(t)

	privKey, _, _ := crypto.GenerateKeyPair()
	validatorManager := MockValidatorManager{PrivKey: privKey}

	store := kvstore.NewKVStore(backend.NewMemDatabase())
	root := core.CreateTestBlock("a0", "")
	root.ChainID = "testchain"
	root.Epoch = 0
	chain := blockchain.NewChain("testchain", store, root)

	ce := NewConsensusEngine(nil, store, chain, nil, validatorManager)

	b1 := core.NewBlock()
	b1.ChainID = chain.ChainID
	b1.Height = chain.Root().Height + 1
	b1.Epoch = 1
	b1.Parent = chain.Root().Hash()
	b1.HCC = b1.Parent
	b1.Proposer = privKey.PublicKey().Address()
	b1.Timestamp = big.NewInt(time.Now().Unix())
	b1.Signature, _ = privKey.Sign(b1.SignBytes())
	eb1, err := chain.AddBlock(b1)
	require.Nil(err)
	eb1 = chain.MarkBlockValid(eb1.Hash())

	b2 := core.NewBlock()
	b2.ChainID = chain.ChainID
	b2.Height = 2
	b2.Epoch = 2
	b2.Parent = b1.Hash()
	b2.HCC = b2.Parent
	b2.Proposer = privKey.PublicKey().Address()
	b2.Timestamp = big.NewInt(time.Now().Unix())
	b2.Signature, _ = privKey.Sign(b2.SignBytes())
	eb2, err := chain.AddBlock(b2)
	require.Nil(err)
	eb2 = chain.MarkBlockValid(eb2.Hash())
	eb2 = chain.MarkBlockHasValidatorUpdate(eb2.Hash())

	// Validator updating block's child
	b3 := core.NewBlock()
	b3.ChainID = chain.ChainID
	b3.Height = 3
	b3.Epoch = 3
	b3.Parent = b2.Hash()
	b3.HCC = b2.Hash()
	b3.Proposer = privKey.PublicKey().Address()
	b3.Timestamp = big.NewInt(time.Now().Unix())
	b3.Signature, _ = privKey.Sign(b3.SignBytes())
	_, err = chain.AddBlock(b3)
	require.Nil(err)
	chain.MarkBlockValid(b3.Hash())

	b4 := core.NewBlock()
	b4.ChainID = chain.ChainID
	b4.Height = 4
	b4.Epoch = 5
	b4.Parent = b3.Hash()
	b4.HCC = b3.Hash()
	b4.Proposer = privKey.PublicKey().Address()
	b4.Timestamp = big.NewInt(time.Now().Unix())
	b4.Signature, _ = privKey.Sign(b4.SignBytes())
	_, err = chain.AddBlock(b4)
	require.Nil(err)
	eb4 := chain.MarkBlockValid(b4.Hash())

	// Valid b5: HCC link to b4
	b5 := core.NewBlock()
	b5.ChainID = chain.ChainID
	b5.Height = 5
	b5.Epoch = 6
	b5.Parent = b4.Hash()
	b5.HCC = b4.Hash()
	b5.Proposer = privKey.PublicKey().Address()
	b5.Timestamp = big.NewInt(time.Now().Unix())
	b5.Signature, _ = privKey.Sign(b5.SignBytes())
	_, err = chain.AddBlock(b5)
	require.Nil(err)
	require.True(ce.validateBlock(b5, eb4))

	// Valid b5: HCC link to b3
	b5 = core.NewBlock()
	b5.ChainID = chain.ChainID
	b5.Height = 5
	b5.Epoch = 7
	b5.Parent = b4.Hash()
	b5.HCC = b3.Hash()
	b5.Proposer = privKey.PublicKey().Address()
	b5.Timestamp = big.NewInt(time.Now().Unix())
	b5.Signature, _ = privKey.Sign(b5.SignBytes())
	_, err = chain.AddBlock(b5)
	require.Nil(err)
	require.True(ce.validateBlock(b5, eb4))
}
