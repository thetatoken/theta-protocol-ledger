package execution

import (
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/result"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/crypto"
	st "github.com/thetatoken/ukulele/ledger/state"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/store/database/backend"
)

// --------------- Test Utilities --------------- //

type TestConsensusEngine struct {
	privKey *crypto.PrivateKey
}

func (tce *TestConsensusEngine) ID() string                        { return tce.privKey.PublicKey().Address().Hex() }
func (tce *TestConsensusEngine) PrivateKey() *crypto.PrivateKey    { return tce.privKey }
func (tce *TestConsensusEngine) GetTip() *core.ExtendedBlock       { return nil }
func (tce *TestConsensusEngine) GetEpoch() uint32                  { return 100 }
func (tce *TestConsensusEngine) AddMessage(msg interface{})        {}
func (tce *TestConsensusEngine) FinalizedBlocks() chan *core.Block { return nil }

func NewTestConsensusEngine(seed string) *TestConsensusEngine {
	privKey, _, _ := crypto.TEST_GenerateKeyPairWithSeed(seed)
	return &TestConsensusEngine{privKey}
}

type TestValidatorManager struct {
	proposer core.Validator
	valSet   *core.ValidatorSet
}

func (tvm *TestValidatorManager) GetProposerForEpoch(epoch uint32) core.Validator { return tvm.proposer }
func (tvm *TestValidatorManager) GetValidatorSetForEpoch(epoch uint32) *core.ValidatorSet {
	return tvm.valSet
}

func NewTestValidatorManager(proposer core.Validator, valSet *core.ValidatorSet) core.ValidatorManager {
	return &TestValidatorManager{
		proposer: proposer,
		valSet:   valSet,
	}
}

type execTest struct {
	chainID  string
	executor *Executor

	accProposer types.PrivAccount
	accVal2     types.PrivAccount

	accIn  types.PrivAccount
	accOut types.PrivAccount
}

func newExecTest() *execTest {
	et := &execTest{}
	et.reset()

	return et
}

//reset everything. state is empty
func (et *execTest) reset() {
	et.accIn = types.MakeAccWithInitBalance("foo", types.Coins{types.Coin{"GammaWei", 5}, types.Coin{"ThetaWei", 700000}})
	et.accOut = types.MakeAccWithInitBalance("bar", types.Coins{types.Coin{"GammaWei", 5}, types.Coin{"ThetaWei", 700000}})
	et.accProposer = types.MakeAcc("proposer")
	et.accVal2 = types.MakeAcc("val2")

	chainID := "test_chain_id"
	initHeight := uint32(1)
	initRootHash := common.Hash{}
	db := backend.NewMemDatabase()
	ledgerState := st.NewLedgerState(chainID, db)
	ledgerState.ResetState(initHeight, initRootHash)

	consensus := NewTestConsensusEngine("localseed")

	propser := core.NewValidator(et.accProposer.PubKey.ToBytes(), uint64(999))
	val2 := core.NewValidator(et.accVal2.PubKey.ToBytes(), uint64(100))
	valSet := core.NewValidatorSet()
	valSet.AddValidator(propser)
	valSet.AddValidator(val2)
	valMgr := NewTestValidatorManager(propser, valSet)

	executor := NewExecutor(ledgerState, consensus, valMgr)

	et.chainID = chainID
	et.executor = executor
}

func (et *execTest) fastforwardBy(heightIncrement uint32) bool {
	height := et.executor.state.Height()
	rootHash := et.executor.state.Commit()
	et.executor.state.ResetState(height+heightIncrement-1, rootHash)
	return true
}

func (et *execTest) fastforwardTo(targetHeight uint32) bool {
	height := et.executor.state.Height()
	rootHash := et.executor.state.Commit()
	if targetHeight < height+1 {
		return false
	}
	et.executor.state.ResetState(targetHeight, rootHash)
	return true
}

func (et *execTest) signSendTx(tx *types.SendTx, accsIn ...types.PrivAccount) {
	types.SignSendTx(et.chainID, tx, accsIn...)
}

func (et *execTest) state() *st.LedgerState {
	return et.executor.state
}

// returns the final balance and expected balance for input and output accounts
func (et *execTest) execSendTx(tx *types.SendTx, checkTx bool) (res result.Result, inGot, inExp, outGot, outExp types.Coins) {
	initBalIn := et.state().GetAccount(et.accIn.Account.PubKey.Address()).Balance
	initBalOut := et.state().GetAccount(et.accOut.Account.PubKey.Address()).Balance

	if checkTx {
		_, res = et.executor.CheckTx(tx)
	} else {
		_, res = et.executor.ExecuteTx(tx)
	}

	endBalIn := et.state().GetAccount(et.accIn.Account.PubKey.Address()).Balance
	endBalOut := et.state().GetAccount(et.accOut.Account.PubKey.Address()).Balance
	decrBalInExp := tx.Outputs[0].Coins.Plus(types.Coins{tx.Fee}) //expected decrease in balance In
	return res, endBalIn, initBalIn.Minus(decrBalInExp), endBalOut, initBalOut.Plus(tx.Outputs[0].Coins)
}

func (et *execTest) acc2State(accs ...types.PrivAccount) {
	for _, acc := range accs {
		et.executor.state.SetAccount(acc.Account.PubKey.Address(), &acc.Account)
	}
	et.executor.state.Commit()
}
