package execution

import (
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/result"
	"github.com/thetatoken/ukulele/core"
	st "github.com/thetatoken/ukulele/ledger/state"
	"github.com/thetatoken/ukulele/ledger/types"
)

var _ TxExecutor = (*CoinbaseTxExecutor)(nil)

// ------------------------------- Coinbase Transaction -----------------------------------

// CoinbaseTxExecutor implements the TxExecutor interface
type CoinbaseTxExecutor struct {
	state     *st.LedgerState
	consensus core.ConsensusEngine
	valMgr    core.ValidatorManager
}

// NewCoinbaseTxExecutor creates a new instance of CoinbaseTxExecutor
func NewCoinbaseTxExecutor(state *st.LedgerState, consensus core.ConsensusEngine, valMgr core.ValidatorManager) *CoinbaseTxExecutor {
	return &CoinbaseTxExecutor{
		state:     state,
		consensus: consensus,
		valMgr:    valMgr,
	}
}

func (exec *CoinbaseTxExecutor) sanityCheck(chainID string, view *st.StoreView, transaction types.Tx) result.Result {
	tx := transaction.(*types.CoinbaseTx)
	validatorAddresses := getValidatorAddresses(exec.consensus, exec.valMgr)

	// Validate proposer, basic
	res := tx.Proposer.ValidateBasic()
	if res.IsError() {
		return res
	}

	// verify that at most one coinbase transaction is processed for each block
	if view.CoinbaseTransactinProcessed() {
		return result.Error("Another coinbase transaction has been processed for the current block")
	}

	// verify the proposer is one of the validators
	res = isAValidator(tx.Proposer.PubKey, validatorAddresses)
	if res.IsError() {
		return res
	}

	proposerAccount, res := getInput(view, tx.Proposer)
	if res.IsError() {
		return res
	}

	// verify the proposer's signature
	signBytes := tx.SignBytes(chainID)
	if !proposerAccount.PubKey.VerifySignature(signBytes, tx.Proposer.Signature) {
		return result.Error("SignBytes: %X", signBytes)
	}

	outputAccounts := map[string]*types.Account{}
	outputAccounts, res = getOrMakeOutputs(view, outputAccounts, tx.Outputs)
	if res.IsError() {
		return res
	}

	if tx.BlockHeight != exec.state.Height() {
		return result.Error("invalid block height for the coinbase transaction, tx_block_height = %v, state_height = %v",
			tx.BlockHeight, exec.state.Height())
	}

	// check the reward amount
	expectedRewards := CalculateReward(view, validatorAddresses)
	if len(expectedRewards) != len(tx.Outputs) {
		return result.Error("Number of rewarded account is incorrect")
	}
	for _, output := range tx.Outputs {
		exp, ok := expectedRewards[string(output.Address[:])]
		if !ok || !exp.IsEqual(output.Coins) {
			return result.Error("Invalid rewards, address %v expecting %v, but is %v",
				output.Address, exp, output.Coins)
		}
	}
	return result.OK
}

func (exec *CoinbaseTxExecutor) process(chainID string, view *st.StoreView, transaction types.Tx) (common.Hash, result.Result) {
	tx := transaction.(*types.CoinbaseTx)

	if view.CoinbaseTransactinProcessed() {
		return common.Hash{}, result.Error("Another coinbase transaction has been processed for the current block")
	}

	accounts := map[string]*types.Account{}
	accounts, res := getOrMakeOutputs(view, accounts, tx.Outputs)
	if res.IsError() {
		return common.Hash{}, res
	}

	for _, output := range tx.Outputs {
		addr := string(output.Address[:])
		if account, exists := accounts[addr]; exists {
			account.Balance = account.Balance.Plus(output.Coins)
			view.SetAccount(output.Address, account)
		}
	}

	view.SetCoinbaseTransactionProcessed(true)

	txHash := types.TxID(chainID, tx)
	return txHash, result.OK
}

// CalculateReward calculates the block reward for each account
func CalculateReward(view *st.StoreView, validatorAddresses []common.Address) map[string]types.Coins {
	accountReward := map[string]types.Coins{}

	for _, validatorAddress := range validatorAddresses {
		// Initial Mainnet release should not reward the validators until the guardians ready to deploy
		zeroReward := types.Coins{}.NoNil()
		accountReward[string(validatorAddress[:])] = zeroReward
	}

	return accountReward
}
