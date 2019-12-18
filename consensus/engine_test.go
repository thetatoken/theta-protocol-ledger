package consensus

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thetatoken/theta/blockchain"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/crypto"
	"github.com/thetatoken/theta/store/database/backend"
	"github.com/thetatoken/theta/store/kvstore"
)

type MockValidatorManager struct {
	PrivKey *crypto.PrivateKey
}

func (m MockValidatorManager) GetProposer(_ common.Hash, _ uint64) core.Validator {
	stake := big.NewInt(10000)
	return core.NewValidator(m.PrivKey.PublicKey().Address().Hex(), stake)
}

func (m MockValidatorManager) GetNextProposer(a common.Hash, b uint64) core.Validator {
	return m.GetProposer(a, b)
}

func (m MockValidatorManager) GetValidatorSet(_ common.Hash) *core.ValidatorSet {
	v := core.NewValidatorSet()
	v.AddValidator(m.GetProposer(common.Hash{}, 0))
	return v
}

func (m MockValidatorManager) GetNextValidatorSet(a common.Hash) *core.ValidatorSet {
	return m.GetNextValidatorSet(a)
}

func (m MockValidatorManager) SetConsensusEngine(consensus core.ConsensusEngine) {}

func TestSingleBlockValidation(t *testing.T) {
	require := require.New(t)

	privKey, _, _ := crypto.GenerateKeyPair()
	addr := privKey.PublicKey().Address()
	validatorManager := MockValidatorManager{PrivKey: privKey}

	store := kvstore.NewKVStore(backend.NewMemDatabase())
	root := core.CreateTestBlock("a0", "")
	root.ChainID = "testchain"
	root.Epoch = 0
	chain := blockchain.NewChain("testchain", store, root)

	ce := NewConsensusEngine(privKey, store, chain, nil, validatorManager)

	// Valid block
	b1 := core.NewBlock()
	b1.ChainID = chain.ChainID
	b1.Height = chain.Root().Height + 1
	b1.Epoch = 1
	b1.Parent = chain.Root().Hash()

	b1.HCC.BlockHash = b1.Parent
	vote := core.Vote{Block: b1.Parent, ID: addr}
	vote.Sign(privKey)
	voteset := core.NewVoteSet()
	voteset.AddVote(vote)
	b1.HCC = core.CommitCertificate{Votes: voteset, BlockHash: b1.Parent}

	b1.Proposer = addr
	b1.Timestamp = big.NewInt(time.Now().Unix())
	b1.Signature, _ = privKey.Sign(b1.SignBytes())
	chain.AddBlock(b1)

	res := ce.validateBlock(b1, chain.Root())
	require.True(res.IsOK())

	// Invalid blocks.
	invalidBlock := core.NewBlock()
	invalidBlock.ChainID = chain.ChainID
	invalidBlock.Epoch = 2
	invalidBlock.Parent = chain.Root().Hash()

	invalidBlock.HCC.BlockHash = invalidBlock.Parent
	vote = core.Vote{Block: invalidBlock.Parent, ID: addr}
	vote.Sign(privKey)
	voteset = core.NewVoteSet()
	voteset.AddVote(vote)
	invalidBlock.HCC = core.CommitCertificate{Votes: voteset, BlockHash: invalidBlock.Parent}

	invalidBlock.Timestamp = big.NewInt(time.Now().Unix())
	invalidBlock.Proposer = privKey.PublicKey().Address()
	invalidBlock.Signature, _ = privKey.Sign(invalidBlock.SignBytes())
	_, err := chain.AddBlock(invalidBlock)
	require.Nil(err)
	invalidBlock.Height = chain.Root().Height + 1
	res = ce.validateBlock(invalidBlock, chain.Root())
	require.True(res.IsOK(), "Should be valid")
	invalidBlock.Height = 0
	invalidBlock.Signature, _ = privKey.Sign(invalidBlock.SignBytes())
	res = ce.validateBlock(invalidBlock, chain.Root())
	require.True(res.IsError(), "Missing height")
	require.Equal("Block is older than last finalized block", res.Message)

	invalidBlock = core.NewBlock()
	invalidBlock.ChainID = chain.ChainID
	invalidBlock.Height = 1
	invalidBlock.Parent = chain.Root().Hash()

	vote = core.Vote{Block: invalidBlock.Parent, ID: addr}
	vote.Sign(privKey)
	voteset = core.NewVoteSet()
	voteset.AddVote(vote)
	invalidBlock.HCC = core.CommitCertificate{Votes: voteset, BlockHash: invalidBlock.Parent}

	invalidBlock.Timestamp = big.NewInt(time.Now().Unix())
	invalidBlock.Proposer = privKey.PublicKey().Address()
	invalidBlock.Signature, _ = privKey.Sign(invalidBlock.SignBytes())
	_, err = chain.AddBlock(invalidBlock)
	require.Nil(err)
	res = ce.validateBlock(invalidBlock, chain.Root())
	require.True(res.IsError(), "Missing epoch")
	require.Equal("Block epoch must be greater than parent epoch", res.Message)

	invalidBlock = core.NewBlock()
	invalidBlock.ChainID = chain.ChainID
	invalidBlock.Height = 1
	invalidBlock.Epoch = 3
	invalidBlock.Parent = common.Hash{}

	vote = core.Vote{Block: invalidBlock.Parent, ID: addr}
	vote.Sign(privKey)
	voteset = core.NewVoteSet()
	voteset.AddVote(vote)
	invalidBlock.HCC = core.CommitCertificate{Votes: voteset, BlockHash: chain.Root().Hash()}

	invalidBlock.Timestamp = big.NewInt(time.Now().Unix())
	invalidBlock.Proposer = privKey.PublicKey().Address()
	invalidBlock.Signature, _ = privKey.Sign(invalidBlock.SignBytes())
	_, err = chain.AddBlock(invalidBlock)
	require.Nil(err)
	res = ce.validateBlock(invalidBlock, chain.Root())
	require.True(res.IsError(), "Missing parent")
	require.Equal("HCC is not ancestor", res.Message)

	invalidBlock = core.NewBlock()
	invalidBlock.ChainID = chain.ChainID
	invalidBlock.Height = 1
	invalidBlock.Epoch = 4
	invalidBlock.Parent = chain.Root().Hash()

	vote = core.Vote{Block: invalidBlock.Parent, ID: addr}
	voteset = core.NewVoteSet()
	voteset.AddVote(vote)
	invalidBlock.HCC = core.CommitCertificate{Votes: voteset, BlockHash: common.Hash{}}

	invalidBlock.Timestamp = big.NewInt(time.Now().Unix())
	invalidBlock.Proposer = privKey.PublicKey().Address()
	invalidBlock.Signature, _ = privKey.Sign(invalidBlock.SignBytes())
	_, err = chain.AddBlock(invalidBlock)
	require.Nil(err)
	res = ce.validateBlock(invalidBlock, chain.Root())
	require.True(res.IsError(), "Missing HCC")
	require.Equal("HCC block not found", res.Message)

	invalidBlock = core.NewBlock()
	invalidBlock.ChainID = chain.ChainID
	invalidBlock.Height = 1
	invalidBlock.Epoch = 4
	invalidBlock.Parent = chain.Root().Hash()

	vote = core.Vote{Block: common.HexToHash("a0b1"), ID: addr}
	vote.Sign(privKey)
	voteset = core.NewVoteSet()
	voteset.AddVote(vote)
	invalidBlock.HCC = core.CommitCertificate{Votes: voteset, BlockHash: common.HexToHash("a0b1")}

	invalidBlock.Timestamp = big.NewInt(time.Now().Unix())
	invalidBlock.Proposer = privKey.PublicKey().Address()
	invalidBlock.Signature, _ = privKey.Sign(invalidBlock.SignBytes())
	_, err = chain.AddBlock(invalidBlock)
	require.Nil(err)
	res = ce.validateBlock(invalidBlock, chain.Root())
	require.True(res.IsError(), "Invalid HCC")
	require.Equal("HCC is not ancestor", res.Message)

	invalidBlock = core.NewBlock()
	invalidBlock.ChainID = chain.ChainID
	invalidBlock.Height = 1
	invalidBlock.Epoch = 5
	invalidBlock.Parent = chain.Root().Hash()

	vote = core.Vote{Block: invalidBlock.Parent, ID: addr}
	vote.Sign(privKey)
	voteset = core.NewVoteSet()
	voteset.AddVote(vote)
	invalidBlock.HCC = core.CommitCertificate{Votes: voteset, BlockHash: invalidBlock.Parent}

	invalidBlock.Timestamp = big.NewInt(time.Now().Unix())
	invalidBlock.Proposer = privKey.PublicKey().Address()
	invalidBlock.Signature, _ = privKey.Sign(invalidBlock.SignBytes())
	_, err = chain.AddBlock(invalidBlock)
	require.Nil(err)
	res = ce.validateBlock(invalidBlock, chain.Root())
	require.Equal("", res.Message)
	require.True(res.IsOK())

	invalidBlock.Proposer = common.Address{}
	res = ce.validateBlock(invalidBlock, chain.Root())
	require.True(res.IsError(), "Missing Proposer")
	require.Equal("Invalid proposer", res.Message)
}

func TestValidParent(t *testing.T) {
	require := require.New(t)

	privKey, _, _ := crypto.GenerateKeyPair()
	addr := privKey.PublicKey().Address()
	validatorManager := MockValidatorManager{PrivKey: privKey}

	store := kvstore.NewKVStore(backend.NewMemDatabase())
	root := core.CreateTestBlock("a0", "")
	root.ChainID = "testchain"
	root.Epoch = 0
	chain := blockchain.NewChain("testchain", store, root)

	ce := NewConsensusEngine(privKey, store, chain, nil, validatorManager)

	b1 := core.NewBlock()
	b1.ChainID = chain.ChainID
	b1.Height = chain.Root().Height + 1
	b1.Epoch = 1
	b1.Parent = chain.Root().Hash()

	vote := core.Vote{Block: b1.Parent, ID: addr}
	vote.Sign(privKey)
	voteset := core.NewVoteSet()
	voteset.AddVote(vote)
	b1.HCC = core.CommitCertificate{Votes: voteset, BlockHash: b1.Parent}

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

	vote = core.Vote{Block: b2.Parent, ID: addr}
	vote.Sign(privKey)
	voteset = core.NewVoteSet()
	voteset.AddVote(vote)
	b2.HCC = core.CommitCertificate{Votes: voteset, BlockHash: b2.Parent}

	b2.Proposer = privKey.PublicKey().Address()
	b2.Timestamp = big.NewInt(time.Now().Unix())
	b2.Signature, _ = privKey.Sign(b2.SignBytes())
	eb2, err := chain.AddBlock(b2)
	require.Nil(err)

	eb1 = chain.MarkBlockInvalid(eb1.Hash())
	res := ce.validateBlock(b2, eb1)
	require.True(res.IsError(), "Parent block is invalid")
	require.Equal("Parent block is invalid", res.Message)

	// HCC: b1 <= b2
	eb1 = chain.MarkBlockValid(eb1.Hash())
	res = ce.validateBlock(b2, eb1)
	require.True(res.IsOK(), "Parent block is valid")

	// HCC: b1 <= b3
	b3 := core.NewBlock()
	b3.ChainID = chain.ChainID
	b3.Height = 3
	b3.Epoch = 3
	b3.Parent = b2.Hash()

	vote = core.Vote{Block: b1.Hash(), ID: addr}
	vote.Sign(privKey)
	voteset = core.NewVoteSet()
	voteset.AddVote(vote)
	b3.HCC = core.CommitCertificate{Votes: voteset, BlockHash: b1.Hash()}

	b3.Proposer = privKey.PublicKey().Address()
	b3.Timestamp = big.NewInt(time.Now().Unix())
	b3.Signature, _ = privKey.Sign(b3.SignBytes())
	_, err = chain.AddBlock(b3)
	require.Nil(err)
	eb2 = chain.MarkBlockValid(eb2.Hash())
	res = ce.validateBlock(b3, eb2)
	require.True(res.IsOK(), "HCC is valid")
}

func TestChildBlockOfValidatorChange(t *testing.T) {
	require := require.New(t)

	privKey, _, _ := crypto.GenerateKeyPair()
	addr := privKey.PublicKey().Address()
	validatorManager := MockValidatorManager{PrivKey: privKey}

	store := kvstore.NewKVStore(backend.NewMemDatabase())
	root := core.CreateTestBlock("a0", "")
	root.ChainID = "testchain"
	root.Epoch = 0
	chain := blockchain.NewChain("testchain", store, root)

	ce := NewConsensusEngine(privKey, store, chain, nil, validatorManager)

	b1 := core.NewBlock()
	b1.ChainID = chain.ChainID
	b1.Height = chain.Root().Height + 1
	b1.Epoch = 1
	b1.Parent = chain.Root().Hash()

	vote := core.Vote{Block: b1.Parent, ID: addr}
	vote.Sign(privKey)
	voteset := core.NewVoteSet()
	voteset.AddVote(vote)
	b1.HCC = core.CommitCertificate{Votes: voteset, BlockHash: b1.Parent}

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

	vote = core.Vote{Block: b2.Parent, ID: addr}
	vote.Sign(privKey)
	voteset = core.NewVoteSet()
	voteset.AddVote(vote)
	b2.HCC = core.CommitCertificate{Votes: voteset, BlockHash: b2.Parent}

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
	vote = core.Vote{Block: b1.Hash(), ID: addr}
	vote.Sign(privKey)
	voteset = core.NewVoteSet()
	voteset.AddVote(vote)
	b3.HCC = core.CommitCertificate{Votes: voteset, BlockHash: b1.Hash()}

	b3.Proposer = privKey.PublicKey().Address()
	b3.Timestamp = big.NewInt(time.Now().Unix())
	b3.Signature, _ = privKey.Sign(b3.SignBytes())
	_, err = chain.AddBlock(b3)
	require.Nil(err)
	res := ce.validateBlock(b3, eb2)
	require.True(res.IsOK(), "HCC is valid")

	// b2 is now marked to have validator changes.
	eb2 = chain.MarkBlockHasValidatorUpdate(eb2.Hash())
	res = ce.validateBlock(b3, eb2)
	require.True(res.IsError(), "Block with validator update need to be its child's HCC")
	require.Equal("HCC incorrect: parent has validator changes", res.Message)

	// Validator updating block's child.
	b3 = core.NewBlock()
	b3.ChainID = chain.ChainID
	b3.Height = 3
	b3.Epoch = 4
	b3.Parent = b2.Hash()

	vote = core.Vote{Block: b3.Parent, ID: addr}
	vote.Sign(privKey)
	voteset = core.NewVoteSet()
	voteset.AddVote(vote)
	b3.HCC = core.CommitCertificate{Votes: voteset, BlockHash: b3.Parent}

	b3.Proposer = privKey.PublicKey().Address()
	b3.Timestamp = big.NewInt(time.Now().Unix())
	b3.Signature, _ = privKey.Sign(b3.SignBytes())
	_, err = chain.AddBlock(b3)
	require.Nil(err)
	res = ce.validateBlock(b3, eb2)
	require.True(res.IsOK(), "HCC is valid")
}

func TestGrandChildBlockOfValidatorChange(t *testing.T) {
	require := require.New(t)

	privKey, _, _ := crypto.GenerateKeyPair()
	addr := privKey.PublicKey().Address()
	validatorManager := MockValidatorManager{PrivKey: privKey}

	store := kvstore.NewKVStore(backend.NewMemDatabase())
	root := core.CreateTestBlock("a0", "")
	root.ChainID = "testchain"
	root.Epoch = 0
	chain := blockchain.NewChain("testchain", store, root)

	ce := NewConsensusEngine(privKey, store, chain, nil, validatorManager)

	b1 := core.NewBlock()
	b1.ChainID = chain.ChainID
	b1.Height = chain.Root().Height + 1
	b1.Epoch = 1
	b1.Parent = chain.Root().Hash()

	vote := core.Vote{Block: b1.Parent, ID: addr}
	vote.Sign(privKey)
	voteset := core.NewVoteSet()
	voteset.AddVote(vote)
	b1.HCC = core.CommitCertificate{Votes: voteset, BlockHash: b1.Parent}

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

	vote = core.Vote{Block: b2.Parent, ID: addr}
	vote.Sign(privKey)
	voteset = core.NewVoteSet()
	voteset.AddVote(vote)
	b2.HCC = core.CommitCertificate{Votes: voteset, BlockHash: b2.Parent}

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

	vote = core.Vote{Block: b3.Parent, ID: addr}
	vote.Sign(privKey)
	voteset = core.NewVoteSet()
	voteset.AddVote(vote)
	b3.HCC = core.CommitCertificate{Votes: voteset, BlockHash: b3.Parent}

	b3.Proposer = privKey.PublicKey().Address()
	b3.Timestamp = big.NewInt(time.Now().Unix())
	b3.Signature, _ = privKey.Sign(b3.SignBytes())
	_, err = chain.AddBlock(b3)
	require.Nil(err)
	eb3 := chain.MarkBlockValid(b3.Hash())

	// No votes in grand child.
	b4 := core.NewBlock()
	b4.ChainID = chain.ChainID
	b4.Height = 4
	b4.Epoch = 5
	b4.Parent = b3.Hash()

	vote = core.Vote{Block: b4.Parent, ID: addr}
	vote.Sign(privKey)
	voteset = core.NewVoteSet()
	b4.HCC = core.CommitCertificate{Votes: voteset, BlockHash: b4.Parent}

	b4.Proposer = privKey.PublicKey().Address()
	b4.Timestamp = big.NewInt(time.Now().Unix())
	b4.Signature, _ = privKey.Sign(b4.SignBytes())
	_, err = chain.AddBlock(b4)
	require.Nil(err)
	res := ce.validateBlock(b4, eb3)
	require.True(res.IsError(), "HCC has no votes")
	require.Equal("Invalid HCC", res.Message)

	// Valid grand child.
	b4 = core.NewBlock()
	b4.ChainID = chain.ChainID
	b4.Height = 4
	b4.Epoch = 5
	b4.Parent = b3.Hash()

	vote = core.Vote{Block: b4.Parent, ID: addr}
	vote.Sign(privKey)
	voteset = core.NewVoteSet()
	voteset.AddVote(vote)
	b4.HCC = core.CommitCertificate{Votes: voteset, BlockHash: b4.Parent}

	b4.Proposer = privKey.PublicKey().Address()
	b4.Timestamp = big.NewInt(time.Now().Unix())
	b4.Signature, _ = privKey.Sign(b4.SignBytes())
	_, err = chain.AddBlock(b4)
	require.Nil(err)
	res = ce.validateBlock(b4, eb3)
	require.True(res.IsOK(), "HCC is valid")

	// Invalid grand child: HCC link to b2.
	b4 = core.NewBlock()
	b4.ChainID = chain.ChainID
	b4.Height = 4
	b4.Epoch = 6
	b4.Parent = b3.Hash()

	vote = core.Vote{Block: b2.Hash(), ID: addr}
	vote.Sign(privKey)
	voteset = core.NewVoteSet()
	voteset.AddVote(vote)
	b4.HCC = core.CommitCertificate{Votes: voteset, BlockHash: b2.Hash()}

	b4.Proposer = privKey.PublicKey().Address()
	b4.Timestamp = big.NewInt(time.Now().Unix())
	b4.Signature, _ = privKey.Sign(b4.SignBytes())
	_, err = chain.AddBlock(b4)
	require.Nil(err)
	res = ce.validateBlock(b4, eb3)
	require.True(res.IsError(), "HCC is valid")
	require.Equal("HCC incorrect: grandparent has validator changes", res.Message)

	// Invalid grand child: HCC link to b1.
	b4 = core.NewBlock()
	b4.ChainID = chain.ChainID
	b4.Height = 4
	b4.Epoch = 7
	b4.Parent = b3.Hash()

	vote = core.Vote{Block: b1.Hash(), ID: addr}
	vote.Sign(privKey)
	voteset = core.NewVoteSet()
	voteset.AddVote(vote)
	b4.HCC = core.CommitCertificate{Votes: voteset, BlockHash: b1.Hash()}

	b4.Proposer = privKey.PublicKey().Address()
	b4.Timestamp = big.NewInt(time.Now().Unix())
	b4.Signature, _ = privKey.Sign(b4.SignBytes())
	_, err = chain.AddBlock(b4)
	require.Nil(err)
	res = ce.validateBlock(b4, eb3)
	require.True(res.IsError(), "HCC is valid")
	require.Equal("HCC incorrect: grandparent has validator changes", res.Message)
}

func TestGrandGrandChildBlockOfValidatorChange(t *testing.T) {
	require := require.New(t)

	privKey, _, _ := crypto.GenerateKeyPair()
	addr := privKey.PublicKey().Address()
	validatorManager := MockValidatorManager{PrivKey: privKey}

	store := kvstore.NewKVStore(backend.NewMemDatabase())
	root := core.CreateTestBlock("a0", "")
	root.ChainID = "testchain"
	root.Epoch = 0
	chain := blockchain.NewChain("testchain", store, root)

	ce := NewConsensusEngine(privKey, store, chain, nil, validatorManager)

	b1 := core.NewBlock()
	b1.ChainID = chain.ChainID
	b1.Height = chain.Root().Height + 1
	b1.Epoch = 1
	b1.Parent = chain.Root().Hash()

	vote := core.Vote{Block: b1.Parent, ID: addr}
	vote.Sign(privKey)
	voteset := core.NewVoteSet()
	voteset.AddVote(vote)
	b1.HCC = core.CommitCertificate{Votes: voteset, BlockHash: b1.Parent}

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

	vote = core.Vote{Block: b2.Parent, ID: addr}
	vote.Sign(privKey)
	voteset = core.NewVoteSet()
	voteset.AddVote(vote)
	b2.HCC = core.CommitCertificate{Votes: voteset, BlockHash: b2.Parent}

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

	vote = core.Vote{Block: b3.Parent, ID: addr}
	vote.Sign(privKey)
	voteset = core.NewVoteSet()
	voteset.AddVote(vote)
	b3.HCC = core.CommitCertificate{Votes: voteset, BlockHash: b3.Parent}

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

	vote = core.Vote{Block: b4.Parent, ID: addr}
	vote.Sign(privKey)
	voteset = core.NewVoteSet()
	voteset.AddVote(vote)
	b4.HCC = core.CommitCertificate{Votes: voteset, BlockHash: b4.Parent}

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

	vote = core.Vote{Block: b5.Parent, ID: addr}
	vote.Sign(privKey)
	voteset = core.NewVoteSet()
	voteset.AddVote(vote)
	b5.HCC = core.CommitCertificate{Votes: voteset, BlockHash: b5.Parent}

	b5.Proposer = privKey.PublicKey().Address()
	b5.Timestamp = big.NewInt(time.Now().Unix())
	b5.Signature, _ = privKey.Sign(b5.SignBytes())
	_, err = chain.AddBlock(b5)
	require.Nil(err)
	res := ce.validateBlock(b5, eb4)
	require.True(res.IsOK())

	// Valid b5: HCC link to b3
	b5 = core.NewBlock()
	b5.ChainID = chain.ChainID
	b5.Height = 5
	b5.Epoch = 7
	b5.Parent = b4.Hash()

	vote = core.Vote{Block: b3.Hash(), ID: addr}
	vote.Sign(privKey)
	voteset = core.NewVoteSet()
	voteset.AddVote(vote)
	b5.HCC = core.CommitCertificate{Votes: voteset, BlockHash: b3.Hash()}

	b5.Proposer = privKey.PublicKey().Address()
	b5.Timestamp = big.NewInt(time.Now().Unix())
	b5.Signature, _ = privKey.Sign(b5.SignBytes())
	_, err = chain.AddBlock(b5)
	require.Nil(err)
	res = ce.validateBlock(b5, eb4)
	require.True(res.IsOK())
}

func TestTipSelection(t *testing.T) {
	assert := assert.New(t)

	privKey, _, _ := crypto.GenerateKeyPair()
	validatorManager := MockValidatorManager{PrivKey: privKey}

	core.ResetTestBlocks()

	store := kvstore.NewKVStore(backend.NewMemDatabase())
	root := core.CreateTestBlock("root", "")
	chain := blockchain.NewChain("testchain", store, root)

	ce := NewConsensusEngine(privKey, store, chain, nil, validatorManager)

	a1 := core.CreateTestBlock("a1", "root")
	chain.AddBlock(a1)

	a2 := core.CreateTestBlock("a2", "a1")
	chain.AddBlock(a2)

	b1 := core.CreateTestBlock("b1", "root")
	chain.AddBlock(b1)

	b2 := core.CreateTestBlock("b2", "b1")
	chain.AddBlock(b2)

	b3 := core.CreateTestBlock("b3", "b2")
	chain.AddBlock(b3)

	tip := ce.GetTipToVote()
	assert.Equal(root.Hash(), tip.Hash(), "should not select invalid blocks")

	chain.MarkBlockValid(a1.Hash())
	chain.MarkBlockValid(a2.Hash())
	chain.MarkBlockValid(b1.Hash())
	chain.MarkBlockValid(b2.Hash())
	chain.MarkBlockValid(b3.Hash())

	tip = ce.GetTipToVote()
	assert.Equal(b3.Hash(), tip.Hash(), "should select longest branch")

	chain.MarkBlockHasValidatorUpdate(b2.Hash())
	tip = ce.GetTipToExtend()
	assert.Equal(a2.Hash(), tip.Hash(), "should not select blocks with validator update that are higher than local HCC")
}
