package execution

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"strconv"

	log "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/blockchain"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/result"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/crypto"
	st "github.com/thetatoken/theta/ledger/state"

	"github.com/thetatoken/theta/ledger/types"
	"github.com/thetatoken/theta/store/database/backend"
)

// --------------- Test Utilities with Mocked Consensus Engine --------------- //

type TestConsensusEngine struct {
	privKey *crypto.PrivateKey
}

func (tce *TestConsensusEngine) ID() string                        { return tce.privKey.PublicKey().Address().Hex() }
func (tce *TestConsensusEngine) PrivateKey() *crypto.PrivateKey    { return tce.privKey }
func (tce *TestConsensusEngine) GetTip(bool) *core.ExtendedBlock   { return nil }
func (tce *TestConsensusEngine) GetEpoch() uint64                  { return 100 }
func (tce *TestConsensusEngine) AddMessage(msg interface{})        {}
func (tce *TestConsensusEngine) FinalizedBlocks() chan *core.Block { return nil }
func (tce *TestConsensusEngine) GetLedger() core.Ledger            { return nil }
func (tce *TestConsensusEngine) GetLastFinalizedBlock() *core.ExtendedBlock {
	return &core.ExtendedBlock{}
}

func NewTestConsensusEngine(seed string) *TestConsensusEngine {
	privKey, _, _ := crypto.TEST_GenerateKeyPairWithSeed(seed)
	return &TestConsensusEngine{privKey}
}

type TestValidatorManager struct {
	proposer core.Validator
	valSet   *core.ValidatorSet
}

func (tvm *TestValidatorManager) SetConsensusEngine(consensus core.ConsensusEngine) {}

func (tvm *TestValidatorManager) GetProposer(blockHash common.Hash, epoch uint64) core.Validator {
	return tvm.proposer
}

func (tvm *TestValidatorManager) GetNextProposer(blockHash common.Hash, epoch uint64) core.Validator {
	return tvm.proposer
}

func (tvm *TestValidatorManager) GetValidatorSet(blockHash common.Hash) *core.ValidatorSet {
	return tvm.valSet
}

func (tvm *TestValidatorManager) GetNextValidatorSet(blockHash common.Hash) *core.ValidatorSet {
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

func NewExecTest() *execTest {
	et := &execTest{}
	et.reset()

	return et
}

//reset everything. state is empty
func (et *execTest) reset() {
	et.accIn = types.MakeAccWithInitBalance("foo", types.NewCoins(700000, 50*getMinimumTxFee()))
	et.accOut = types.MakeAccWithInitBalance("bar", types.NewCoins(700000, 50*getMinimumTxFee()))
	et.accProposer = types.MakeAcc("proposer")
	et.accVal2 = types.MakeAcc("val2")

	chainID := "test_chain_id"
	initHeight := uint64(1)
	initRootHash := common.Hash{}
	initBlock := &core.Block{
		BlockHeader: &core.BlockHeader{
			ChainID:   chainID,
			Height:    initHeight,
			StateHash: initRootHash,
		},
	}
	db := backend.NewMemDatabase()
	ledgerState := st.NewLedgerState(chainID, db, nil)
	//ledgerState.ResetState(initHeight, initRootHash)
	ledgerState.ResetState(initBlock)

	consensus := NewTestConsensusEngine("localseed")

	propser := core.NewValidator(et.accProposer.PrivKey.PublicKey().Address().String(), new(big.Int).SetUint64(999))
	val2 := core.NewValidator(et.accVal2.PrivKey.PublicKey().Address().String(), new(big.Int).SetUint64(100))
	valSet := core.NewValidatorSet()
	valSet.AddValidator(propser)
	valSet.AddValidator(val2)
	valMgr := NewTestValidatorManager(propser, valSet)

	chain := blockchain.CreateTestChain()
	executor := NewExecutor(db, chain, ledgerState, consensus, valMgr, nil)

	et.chainID = chainID
	et.executor = executor
}

func (et *execTest) fastforwardBy(heightIncrement uint64) bool {
	height := et.executor.state.Height()
	incrementedHeight := height + heightIncrement - 1
	rootHash := et.executor.state.Commit()
	block := &core.Block{
		BlockHeader: &core.BlockHeader{
			Height:    incrementedHeight,
			StateHash: rootHash,
		},
	}
	//et.executor.state.ResetState(height+heightIncrement-1, rootHash)
	et.executor.state.ResetState(block)
	return true
}

func (et *execTest) fastforwardTo(targetHeight uint64) bool {
	height := et.executor.state.Height()
	rootHash := et.executor.state.Commit()
	if targetHeight < height+1 {
		return false
	}
	block := &core.Block{
		BlockHeader: &core.BlockHeader{
			Height:    targetHeight,
			StateHash: rootHash,
		},
	}
	//et.executor.state.ResetState(targetHeight, rootHash)
	et.executor.state.ResetState(block)
	return true
}

func (et *execTest) signSendTx(tx *types.SendTx, accsIn ...types.PrivAccount) {
	types.SignSendTx(et.chainID, tx, accsIn...)
}

func (et *execTest) state() *st.LedgerState {
	return et.executor.state
}

// returns the final balance and expected balance for input and output accounts
func (et *execTest) execSendTx(tx *types.SendTx, screenTx bool) (res result.Result, inGot, inExp, outGot, outExp types.Coins) {
	initBalIn := et.state().Delivered().GetAccount(et.accIn.Account.Address).Balance
	initBalOut := et.state().Delivered().GetAccount(et.accOut.Account.Address).Balance

	if screenTx {
		_, res = et.executor.ScreenTx(tx)
	} else {
		_, res = et.executor.ExecuteTx(tx)
	}

	endBalIn := et.state().Delivered().GetAccount(et.accIn.Account.Address).Balance
	endBalOut := et.state().Delivered().GetAccount(et.accOut.Account.Address).Balance
	decrBalInExp := tx.Outputs[0].Coins.Plus(tx.Fee) //expected decrease in balance In
	return res, endBalIn, initBalIn.Minus(decrBalInExp), endBalOut, initBalOut.Plus(tx.Outputs[0].Coins)
}

func (et *execTest) acc2State(accs ...types.PrivAccount) {
	for _, acc := range accs {
		et.executor.state.Delivered().SetAccount(acc.Account.Address, &acc.Account)
	}
	et.executor.state.Commit()
}

// Executor returns the executor instance.
func (et *execTest) Executor() *Executor {
	return et.executor
}

// State returns the state instance.
func (et *execTest) State() *st.LedgerState {
	return et.state()
}

// SetAcc saves accounts into state.
func (et *execTest) SetAcc(accs ...types.PrivAccount) {
	et.acc2State(accs...)
}

func getMinimumTxFee() int64 {
	return int64(types.MinimumTransactionFeeTFuelWeiJune2021)
}

func createServicePaymentTx(chainID string, source, target *types.PrivAccount, amount int64, srcSeq, tgtSeq, paymentSeq, reserveSeq int, resourceID string) *types.ServicePaymentTx {
	servicePaymentTx := &types.ServicePaymentTx{
		Fee: types.NewCoins(0, getMinimumTxFee()),
		Source: types.TxInput{
			Address:  source.Address,
			Coins:    types.Coins{TFuelWei: big.NewInt(amount), ThetaWei: big.NewInt(0)},
			Sequence: uint64(srcSeq),
		},
		Target: types.TxInput{
			Address:  target.Address,
			Sequence: uint64(tgtSeq),
		},
		PaymentSequence: uint64(paymentSeq),
		ReserveSequence: uint64(reserveSeq),
		ResourceID:      resourceID,
	}

	srcSignBytes := servicePaymentTx.SourceSignBytes(chainID)
	servicePaymentTx.Source.Signature = source.Sign(srcSignBytes)

	tgtSignBytes := servicePaymentTx.TargetSignBytes(chainID)
	servicePaymentTx.Target.Signature = target.Sign(tgtSignBytes)

	if !servicePaymentTx.Source.Signature.Verify(srcSignBytes, source.Address) {
		panic("Signature verification failed for source")
	}
	if !servicePaymentTx.Target.Signature.Verify(tgtSignBytes, target.Address) {
		panic("Signature verification failed for target")
	}

	return servicePaymentTx
}

func setupForServicePayment(ast *assert.Assertions) (et *execTest, resourceID string,
	alice, bob, carol types.PrivAccount, aliceInitBalance, bobInitBalance, carolInitBalance types.Coins) {
	et = NewExecTest()

	alice = types.MakeAcc("User Alice")
	aliceInitBalance = types.Coins{TFuelWei: big.NewInt(10000 * getMinimumTxFee()), ThetaWei: big.NewInt(0)}
	alice.Balance = aliceInitBalance
	et.acc2State(alice)
	log.Infof("Alice's Address: %v", alice.Address.Hex())

	bob = types.MakeAcc("User Bob")
	bobInitBalance = types.Coins{TFuelWei: big.NewInt(3000 * getMinimumTxFee()), ThetaWei: big.NewInt(0)}
	bob.Balance = bobInitBalance
	et.acc2State(bob)
	log.Infof("Bob's Address: %v", bob.Address.Hex())

	carol = types.MakeAcc("User Carol")
	carolInitBalance = types.Coins{TFuelWei: big.NewInt(3000 * getMinimumTxFee()), ThetaWei: big.NewInt(0)}
	carol.Balance = carolInitBalance
	et.acc2State(carol)
	log.Infof("Carol's Address: %v", carol.Address.Hex())

	et.fastforwardTo(1e2)

	resourceID = "rid001"
	reserveFundTx := &types.ReserveFundTx{
		Fee: types.NewCoins(0, getMinimumTxFee()),
		Source: types.TxInput{
			Address:  alice.Address,
			Coins:    types.Coins{TFuelWei: big.NewInt(1000 * getMinimumTxFee()), ThetaWei: big.NewInt(0)},
			Sequence: 1,
		},
		Collateral:  types.Coins{TFuelWei: big.NewInt(1001 * getMinimumTxFee()), ThetaWei: big.NewInt(0)},
		ResourceIDs: []string{resourceID},
		Duration:    1000,
	}
	reserveFundTx.Source.Signature = alice.Sign(reserveFundTx.SignBytes(et.chainID))
	res := et.executor.getTxExecutor(reserveFundTx).sanityCheck(et.chainID, et.state().Delivered(), core.DeliveredView, reserveFundTx)
	ast.True(res.IsOK(), res.String())
	_, res = et.executor.getTxExecutor(reserveFundTx).process(et.chainID, et.state().Delivered(), core.DeliveredView, reserveFundTx)
	ast.True(res.IsOK(), res.String())

	return et, resourceID, alice, bob, carol, aliceInitBalance, bobInitBalance, carolInitBalance
}

type contractByteCode struct {
	DeploymentCode string `json:"deployment_code"`
	Code           string `json:"code"`
}

func loadJSONTest(file string, val interface{}) error {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(content, val); err != nil {
		if syntaxerr, ok := err.(*json.SyntaxError); ok {
			line := findLine(content, syntaxerr.Offset)
			return fmt.Errorf("JSON syntax error at %v:%v: %v", file, line, err)
		}
		return fmt.Errorf("JSON unmarshal error in %v: %v", file, err)
	}
	return nil
}

func findLine(data []byte, offset int64) (line int) {
	line = 1
	for i, r := range string(data) {
		if int64(i) >= offset {
			return
		}
		if r == '\n' {
			line++
		}
	}
	return
}

func setupForSmartContract(ast *assert.Assertions, numAccounts int) (et *execTest, privAccounts []types.PrivAccount) {
	et = NewExecTest()

	for i := 0; i < numAccounts; i++ {
		secret := "acc_secret_" + strconv.FormatInt(int64(i), 16)
		privAccount := types.MakeAccWithInitBalance(secret,
			types.Coins{
				big.NewInt(0),
				big.NewInt(1).Mul(big.NewInt(9000000), big.NewInt(int64(types.MinimumGasPriceJune2021))),
			})
		privAccounts = append(privAccounts, privAccount)
		et.acc2State(privAccount)
	}
	et.fastforwardTo(1e2)

	return et, privAccounts
}
