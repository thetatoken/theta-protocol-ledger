package ledger

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/result"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/crypto"
	dp "github.com/thetatoken/ukulele/dispatcher"
	exec "github.com/thetatoken/ukulele/ledger/execution"
	"github.com/thetatoken/ukulele/ledger/types"
	mp "github.com/thetatoken/ukulele/mempool"
	"github.com/thetatoken/ukulele/p2p"
	p2psim "github.com/thetatoken/ukulele/p2p/simulation"
	"github.com/thetatoken/ukulele/store/database/backend"
)

func TestLedgerSetup(t *testing.T) {
	assert := assert.New(t)

	_, ledger, mempool := newTestLedger()
	assert.NotNil(ledger)
	assert.NotNil(mempool)
}

func TestLedgerCheckTx(t *testing.T) {
	assert := assert.New(t)

	chainID, ledger, _ := newTestLedger()
	accOut, accIn1, accIn2, accIn3 := prepareInitLedgerState(ledger)

	sendTxBytes := newRawSendTx(chainID, 1, accOut, accIn1, accIn2, accIn3)
	res := ledger.CheckTx(sendTxBytes)
	assert.True(res.IsOK(), res.Message)

	coinbaseTxBytes := newRawCoinbaseTx(chainID, ledger, 1)
	res = ledger.CheckTx(coinbaseTxBytes)
	assert.Equal(result.CodeUnauthorizedTx, res.Code, res.Message)
}

// ----------- Utilities ----------- //

func newTestLedger() (chainID string, ledger *Ledger, mempool *mp.Mempool) {
	chainID = "test_chain_id"
	peerID := "peer0"
	proposerSeed := "proposer"

	db := backend.NewMemDatabase()
	consensus := exec.NewTestConsensusEngine(proposerSeed)
	valMgr := newTesetValidatorManager(consensus)
	p2psimnet := p2psim.NewSimnetWithHandler(nil)
	messenger := p2psimnet.AddEndpoint(peerID)
	mempool = newTestMempool(peerID, messenger)
	ledger = NewLedger(chainID, db, consensus, valMgr, mempool)
	mempool.SetLedger(ledger)

	messenger.Start()
	mempool.Start()

	initHeight := uint32(1)
	initRootHash := common.Hash{}
	ledger.ResetState(initHeight, initRootHash)

	return chainID, ledger, mempool
}

func newTesetValidatorManager(consensus core.ConsensusEngine) core.ValidatorManager {
	proposerPubKeyBytes := consensus.PrivateKey().PublicKey().ToBytes()

	propser := core.NewValidator(proposerPubKeyBytes, uint64(999))
	_, val2PubKey, err := crypto.TEST_GenerateKeyPairWithSeed("val2")
	if err != nil {
		panic(fmt.Sprintf("Failed to generate key pair with seed: %v", err))
	}
	val2 := core.NewValidator(val2PubKey.ToBytes(), uint64(100))

	valSet := core.NewValidatorSet()
	valSet.AddValidator(propser)
	valSet.AddValidator(val2)
	valMgr := exec.NewTestValidatorManager(propser, valSet)

	return valMgr
}

func newTestMempool(peerID string, messenger p2p.Network) *mp.Mempool {
	dispatcher := dp.NewDispatcher(messenger)
	mempool := mp.CreateMempool(dispatcher)
	txMsgHandler := mp.CreateMempoolMessageHandler(mempool)
	messenger.RegisterMessageHandler(txMsgHandler)
	return mempool
}

func prepareInitLedgerState(ledger *Ledger) (accOut, accIn1, accIn2, accIn3 types.PrivAccount) {
	accOut = types.MakeAccWithInitBalance("bar", types.Coins{types.Coin{"GammaWei", 5}, types.Coin{"ThetaWei", 700000}})
	accIn1 = types.MakeAccWithInitBalance("foox", types.Coins{types.Coin{"GammaWei", 50000}, types.Coin{"ThetaWei", 900000}})
	accIn2 = types.MakeAccWithInitBalance("fooy", types.Coins{types.Coin{"GammaWei", 50000}, types.Coin{"ThetaWei", 900000}})
	accIn3 = types.MakeAccWithInitBalance("fooz", types.Coins{types.Coin{"GammaWei", 50000}, types.Coin{"ThetaWei", 900000}})

	accs := []types.PrivAccount{accOut, accIn1, accIn2, accIn3}
	for _, acc := range accs {
		ledger.state.SetAccount(acc.Account.PubKey.Address(), &acc.Account)
	}
	ledger.state.Commit()

	return accOut, accIn1, accIn2, accIn3
}

func newRawCoinbaseTx(chainID string, ledger *Ledger, sequence int) common.Bytes {
	epoch := uint32(100)
	vaList := ledger.valMgr.GetValidatorSetForEpoch(epoch).Validators()
	if len(vaList) < 2 {
		panic("Insufficient number of validators")
	}
	va2 := vaList[1]
	proposerSk := ledger.consensus.PrivateKey()
	proposerPk := proposerSk.PublicKey()
	va2Pk := va2.PublicKey()

	coinbaseTx := &types.CoinbaseTx{
		Proposer: types.TxInput{Address: proposerPk.Address(), PubKey: proposerPk, Sequence: sequence},
		Outputs: []types.TxOutput{
			{proposerPk.Address(), types.Coins{{"ThetaWei", 317}}},
			{(&va2Pk).Address(), types.Coins{{"ThetaWei", 317}}},
		},
		BlockHeight: 49,
	}
	signBytes := coinbaseTx.SignBytes(chainID)
	sig, err := proposerSk.Sign(signBytes)
	if err != nil {
		panic("Failed to sign the coinbase transaction")
	}
	if !coinbaseTx.SetSignature(proposerPk.Address(), sig) {
		panic("Failed to set signature for the coinbase transaction")
	}

	coinbaseTxBytes := types.TxToBytes(coinbaseTx)
	return coinbaseTxBytes
}

func newRawSendTx(chainID string, sequence int, accOut, accIn1, accIn2, accIn3 types.PrivAccount) common.Bytes {
	sendTx := &types.SendTx{
		Gas: 0,
		Fee: types.Coin{"GammaWei", 3},
		Inputs: []types.TxInput{
			{Sequence: sequence, PubKey: accIn1.PubKey, Address: accIn1.PubKey.Address(), Coins: types.Coins{types.Coin{"GammaWei", 1}, types.Coin{"ThetaWei", 4}}},
			{Sequence: sequence, PubKey: accIn2.PubKey, Address: accIn2.PubKey.Address(), Coins: types.Coins{types.Coin{"GammaWei", 2}, types.Coin{"ThetaWei", 2}}},
			{Sequence: sequence, PubKey: accIn3.PubKey, Address: accIn3.PubKey.Address(), Coins: types.Coins{types.Coin{"ThetaWei", 6}}},
		},
		Outputs: []types.TxOutput{{Address: accOut.PubKey.Address(), Coins: types.Coins{{"ThetaWei", 12}}}},
	}

	signBytes := sendTx.SignBytes(chainID)
	inAccs := []types.PrivAccount{accIn1, accIn2, accIn3}
	for idx, in := range sendTx.Inputs {
		inAcc := inAccs[idx]
		sig, err := inAcc.PrivKey.Sign(signBytes)
		if err != nil {
			panic("Failed to sign the coinbase transaction")
		}
		sendTx.SetSignature(in.Address, sig)
	}

	sendTxBytes := types.TxToBytes(sendTx)
	return sendTxBytes
}
