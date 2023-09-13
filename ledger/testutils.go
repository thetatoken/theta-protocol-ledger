package ledger

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"sync"

	"github.com/thetatoken/theta/blockchain"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/consensus"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/crypto"
	dp "github.com/thetatoken/theta/dispatcher"
	exec "github.com/thetatoken/theta/ledger/execution"
	"github.com/thetatoken/theta/ledger/state"
	st "github.com/thetatoken/theta/ledger/state"
	"github.com/thetatoken/theta/ledger/types"
	mp "github.com/thetatoken/theta/mempool"
	"github.com/thetatoken/theta/p2p"
	p2psim "github.com/thetatoken/theta/p2p/simulation"
	"github.com/thetatoken/theta/p2pl"
	"github.com/thetatoken/theta/store/database"
	"github.com/thetatoken/theta/store/database/backend"
	"github.com/thetatoken/theta/store/kvstore"
)

type mockSnapshot struct {
	block *core.Block
	vcp   *core.ValidatorCandidatePool
}

type execSim struct {
	chainID   string
	chain     *blockchain.Chain
	state     *st.LedgerState
	consensus *consensus.ConsensusEngine
	executor  *exec.Executor
}

func newExecSim(chainID string, db database.Database, snapshot mockSnapshot, valPrivAcc *types.PrivAccount) *execSim {
	initHeight := snapshot.block.Height

	sv := state.NewStoreView(initHeight, common.Hash{}, db)
	sv.UpdateValidatorCandidatePool(snapshot.vcp)

	store := kvstore.NewKVStore(db)
	chain := blockchain.NewChain(chainID, store, snapshot.block)

	p2psimnet := p2psim.NewSimnetWithHandler(nil)
	messenger := p2psimnet.AddEndpoint("peerID0")

	dispatcher := dp.NewDispatcher(messenger, nil)

	valMgr := consensus.NewFixedValidatorManager()
	consensus := consensus.NewConsensusEngine(valPrivAcc.PrivKey, store, chain, dispatcher, valMgr)
	valMgr.SetConsensusEngine(consensus)

	mempool := mp.CreateMempool(dispatcher, consensus)

	ledgerState := st.NewLedgerState(chainID, db, nil)
	//ledgerState.ResetState(initHeight, snapshot.block.StateHash)
	ledgerState.ResetState(snapshot.block)

	ledger := &Ledger{
		consensus: consensus,
		valMgr:    valMgr,
		mempool:   mempool,
		mu:        &sync.RWMutex{},
		state:     ledgerState,
	}
	executor := exec.NewExecutor(db, chain, ledgerState, consensus, valMgr, ledger)
	ledger.SetExecutor(executor)

	consensus.SetLedger(ledger)

	es := &execSim{
		chainID:   chainID,
		chain:     chain,
		state:     ledgerState,
		consensus: consensus,
		executor:  executor,
	}

	return es
}

func (es *execSim) addBlock(block *core.Block) {
	es.chain.AddBlock(block)
}

func (es *execSim) finalizePreviousBlocks(blockHash common.Hash) {
	es.chain.FinalizePreviousBlocks(blockHash)
}

func (es *execSim) getTipBlock() *core.ExtendedBlock {
	return es.consensus.GetTip(true)
}

func (es *execSim) findBlocksByHeight(height uint64) []*core.ExtendedBlock {
	return es.chain.FindBlocksByHeight(height)
}

func genSimSnapshot(chainID string, db database.Database) (snapshot mockSnapshot, srcPrivAccs []*types.PrivAccount, valPrivAccs []*types.PrivAccount) {
	initHeight := uint64(0)

	src1Acc := types.MakeAcc("src1")
	src2Acc := types.MakeAcc("src2")
	src3Acc := types.MakeAcc("src3")
	src4Acc := types.MakeAcc("src4")
	src5Acc := types.MakeAcc("src5")
	src6Acc := types.MakeAcc("src6")
	src1Acc.Balance = types.Coins{
		ThetaWei: new(big.Int).Mul(new(big.Int).SetUint64(20), core.MinValidatorStakeDeposit),
		TFuelWei: new(big.Int).Mul(new(big.Int).SetUint64(100), core.MinValidatorStakeDeposit),
	}
	src2Acc.Balance = types.Coins{
		ThetaWei: new(big.Int).Mul(new(big.Int).SetUint64(20), core.MinValidatorStakeDeposit),
		TFuelWei: new(big.Int).Mul(new(big.Int).SetUint64(100), core.MinValidatorStakeDeposit),
	}
	src3Acc.Balance = types.Coins{
		ThetaWei: new(big.Int).Mul(new(big.Int).SetUint64(20), core.MinValidatorStakeDeposit),
		TFuelWei: new(big.Int).Mul(new(big.Int).SetUint64(100), core.MinValidatorStakeDeposit),
	}
	src4Acc.Balance = types.Coins{
		ThetaWei: new(big.Int).Mul(new(big.Int).SetUint64(20), core.MinValidatorStakeDeposit),
		TFuelWei: new(big.Int).Mul(new(big.Int).SetUint64(100), core.MinValidatorStakeDeposit),
	}
	src5Acc.Balance = types.Coins{
		ThetaWei: new(big.Int).Mul(new(big.Int).SetUint64(20), core.MinValidatorStakeDeposit),
		TFuelWei: new(big.Int).Mul(new(big.Int).SetUint64(100), core.MinValidatorStakeDeposit),
	}
	src6Acc.Balance = types.Coins{
		ThetaWei: new(big.Int).Mul(new(big.Int).SetUint64(20), core.MinValidatorStakeDeposit),
		TFuelWei: new(big.Int).Mul(new(big.Int).SetUint64(100), core.MinValidatorStakeDeposit),
	}

	val1Acc := types.MakeAcc("va1")
	val2Acc := types.MakeAcc("va2")
	val3Acc := types.MakeAcc("va3")
	val4Acc := types.MakeAcc("va4")
	val5Acc := types.MakeAcc("va5")
	val6Acc := types.MakeAcc("va6")

	stakeAmount1 := new(big.Int).Mul(new(big.Int).SetUint64(5), core.MinValidatorStakeDeposit)
	stakeAmount2 := new(big.Int).Mul(new(big.Int).SetUint64(6), core.MinValidatorStakeDeposit)
	stakeAmount3 := new(big.Int).Mul(new(big.Int).SetUint64(7), core.MinValidatorStakeDeposit)
	stakeAmount4 := new(big.Int).Mul(new(big.Int).SetUint64(4), core.MinValidatorStakeDeposit)

	vcp := &core.ValidatorCandidatePool{}
	vcp.DepositStake(src1Acc.Address, val1Acc.Address, stakeAmount1, 0)
	vcp.DepositStake(src2Acc.Address, val2Acc.Address, stakeAmount2, 0)
	vcp.DepositStake(src3Acc.Address, val3Acc.Address, stakeAmount3, 0)
	vcp.DepositStake(src4Acc.Address, val4Acc.Address, stakeAmount4, 0)

	sv := state.NewStoreView(initHeight, common.Hash{}, db)
	sv.UpdateValidatorCandidatePool(vcp)

	sv.SetAccount(src1Acc.Address, &src1Acc.Account)
	sv.SetAccount(src2Acc.Address, &src2Acc.Account)
	sv.SetAccount(src3Acc.Address, &src3Acc.Account)
	sv.SetAccount(src4Acc.Address, &src4Acc.Account)
	sv.SetAccount(src5Acc.Address, &src5Acc.Account)
	sv.SetAccount(src6Acc.Address, &src6Acc.Account)

	sv.SetAccount(val1Acc.Address, &val1Acc.Account)
	sv.SetAccount(val2Acc.Address, &val2Acc.Account)
	sv.SetAccount(val3Acc.Address, &val3Acc.Account)
	sv.SetAccount(val4Acc.Address, &val4Acc.Account)
	sv.SetAccount(val5Acc.Address, &val5Acc.Account)
	sv.SetAccount(val6Acc.Address, &val6Acc.Account)

	initStateHash := sv.Save()

	initBlock := core.NewBlock()
	initBlock.ChainID = chainID
	initBlock.BlockHeader.StateHash = initStateHash

	snapshot = mockSnapshot{
		block: initBlock,
		vcp:   vcp,
	}
	srcPrivAccs = []*types.PrivAccount{&src1Acc, &src2Acc, &src3Acc, &src4Acc, &src5Acc, &src6Acc}
	valPrivAccs = []*types.PrivAccount{&val1Acc, &val2Acc, &val3Acc, &val4Acc, &val5Acc, &val6Acc}

	return snapshot, srcPrivAccs, valPrivAccs
}

func newTestLedger() (chainID string, ledger *Ledger, mempool *mp.Mempool) {
	chainID = "test_chain_id"
	peerID := "peer0"
	proposerSeed := "proposer"

	db := backend.NewMemDatabase()
	chain := &blockchain.Chain{ChainID: chainID}
	consensus := exec.NewTestConsensusEngine(proposerSeed)
	valMgr := newTesetValidatorManager(consensus)
	p2psimnet := p2psim.NewSimnetWithHandler(nil)
	messenger := p2psimnet.AddEndpoint(peerID)
	mempool = newTestMempool(peerID, messenger, nil)
	ledger = NewLedger(chainID, db, nil, chain, consensus, valMgr, mempool)
	mempool.SetLedger(ledger)

	ctx := context.Background()
	messenger.Start(ctx)
	mempool.Start(ctx)

	initHeight := uint64(1)
	initRootHash := common.Hash{}

	initBlock := &core.Block{
		BlockHeader: &core.BlockHeader{
			ChainID:   chainID,
			Height:    initHeight,
			StateHash: initRootHash,
		},
	}
	//ledger.ResetState(initHeight, initRootHash)
	ledger.ResetState(initBlock)

	return chainID, ledger, mempool
}

func newTesetValidatorManager(consensus core.ConsensusEngine) core.ValidatorManager {
	proposerAddressStr := consensus.PrivateKey().PublicKey().Address().String()
	propser := core.NewValidator(proposerAddressStr, new(big.Int).SetUint64(999))

	_, val2PubKey, err := crypto.TEST_GenerateKeyPairWithSeed("val2")
	if err != nil {
		panic(fmt.Sprintf("Failed to generate key pair with seed: %v", err))
	}
	val2 := core.NewValidator(val2PubKey.Address().String(), new(big.Int).SetUint64(100))

	valSet := core.NewValidatorSet()
	valSet.AddValidator(propser)
	valSet.AddValidator(val2)
	valMgr := exec.NewTestValidatorManager(propser, valSet)

	return valMgr
}

func newTestMempool(peerID string, messenger p2p.Network, messengerL p2pl.Network) *mp.Mempool {
	dispatcher := dp.NewDispatcher(messenger, nil)
	mempool := mp.CreateMempool(dispatcher, nil)
	txMsgHandler := mp.CreateMempoolMessageHandler(mempool)
	messenger.RegisterMessageHandler(txMsgHandler)
	return mempool
}

func prepareInitLedgerState(ledger *Ledger, numInAccs int) (accOut types.PrivAccount, accIns []types.PrivAccount) {
	txFee := getMinimumTxFee()
	validators := ledger.valMgr.GetValidatorSet(common.Hash{}).Validators()
	for _, val := range validators {
		valAccount := &types.Account{
			Address:                val.Address,
			LastUpdatedBlockHeight: 1,
			Balance:                types.NewCoins(100000000000, 1000),
		}
		ledger.state.Delivered().SetAccount(val.Address, valAccount)
	}

	accOut = types.MakeAccWithInitBalance("accOut", types.NewCoins(700000, 3))
	ledger.state.Delivered().SetAccount(accOut.Account.Address, &accOut.Account)

	for i := 0; i < numInAccs; i++ {
		secret := "in_secret_" + strconv.FormatInt(int64(i), 16)
		accIn := types.MakeAccWithInitBalance(secret, types.NewCoins(900000, 50000*txFee))
		accIns = append(accIns, accIn)
		ledger.state.Delivered().SetAccount(accIn.Account.Address, &accIn.Account)
	}

	ledger.state.Commit()

	return accOut, accIns
}

func newRawCoinbaseTx(chainID string, ledger *Ledger, sequence int) common.Bytes {
	vaList := ledger.valMgr.GetValidatorSet(common.Hash{}).Validators()
	if len(vaList) < 2 {
		panic("Insufficient number of validators")
	}
	outputs := []types.TxOutput{}
	for _, val := range vaList {
		output := types.TxOutput{val.Address, types.NewCoins(0, 0)}
		outputs = append(outputs, output)
	}

	proposerSk := ledger.consensus.PrivateKey()
	proposerPk := proposerSk.PublicKey()
	coinbaseTx := &types.CoinbaseTx{
		Proposer:    types.TxInput{Address: proposerPk.Address(), Sequence: uint64(sequence)},
		Outputs:     outputs,
		BlockHeight: 2,
	}

	signBytes := coinbaseTx.SignBytes(chainID)
	sig, err := proposerSk.Sign(signBytes)
	if err != nil {
		panic("Failed to sign the coinbase transaction")
	}
	if !coinbaseTx.SetSignature(proposerPk.Address(), sig) {
		panic("Failed to set signature for the coinbase transaction")
	}

	coinbaseTxBytes, err := types.TxToBytes(coinbaseTx)
	if err != nil {
		panic(err)
	}
	return coinbaseTxBytes
}

func newRawSendTx(chainID string, sequence int, addPubKey bool, accOut, accIn types.PrivAccount, injectFeeFluctuation bool) common.Bytes {
	delta := int64(0)
	var err error
	if injectFeeFluctuation {
		// inject so fluctuation into the txFee, so later we can test whether the
		// mempool orders the txs by txFee
		randint, err := strconv.ParseInt(string(accIn.Address.Hex()[2:9]), 16, 64)
		if randint < 0 {
			randint = -randint
		}
		delta = randint * int64(types.GasRegularTxJune2021)
		if err != nil {
			panic(err)
		}
	}
	txFee := getMinimumTxFee() + delta
	sendTx := &types.SendTx{
		Fee: types.NewCoins(0, txFee),
		Inputs: []types.TxInput{
			{
				Sequence: uint64(sequence),
				Address:  accIn.Address,
				Coins:    types.NewCoins(15, txFee),
			},
		},
		Outputs: []types.TxOutput{
			{
				Address: accOut.Address,
				Coins:   types.NewCoins(15, 0),
			},
		},
	}

	signBytes := sendTx.SignBytes(chainID)
	inAccs := []types.PrivAccount{accIn}
	for idx, in := range sendTx.Inputs {
		inAcc := inAccs[idx]
		sig, err := inAcc.PrivKey.Sign(signBytes)
		if err != nil {
			panic("Failed to sign the coinbase transaction")
		}
		sendTx.SetSignature(in.Address, sig)
	}

	sendTxBytes, err := types.TxToBytes(sendTx)
	if err != nil {
		panic(err)
	}
	return sendTxBytes
}

func getMinimumTxFee() int64 {
	return int64(types.MinimumTransactionFeeTFuelWei)
}
