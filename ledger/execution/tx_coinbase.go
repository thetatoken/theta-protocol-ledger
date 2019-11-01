package execution

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/result"
	"github.com/thetatoken/theta/core"
	st "github.com/thetatoken/theta/ledger/state"
	"github.com/thetatoken/theta/ledger/types"
)

var weiMultiplier = big.NewInt(1e18)
var tfuelRewardPerBlock = big.NewInt(1).Mul(big.NewInt(48), weiMultiplier) // 48 TFUEL per block, corresponds to about 5% *initial* annual inflation rate. The inflation rate naturally approaches 0 as the chain grows.
var checkpointInterval = int64(100)                                        // TODO: use the guarding checkpoint

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
	validatorSet := getValidatorSet(exec.consensus.GetLedger(), exec.valMgr)
	validatorAddresses := getValidatorAddresses(validatorSet)

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
	res = isAValidator(tx.Proposer.Address, validatorAddresses)
	if res.IsError() {
		return res
	}

	proposerAccount, res := getOrMakeInput(view, tx.Proposer)
	if res.IsError() {
		return res
	}

	// verify the proposer's signature
	signBytes := tx.SignBytes(chainID)
	if !tx.Proposer.Signature.Verify(signBytes, proposerAccount.Address) {
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
	expectedRewards := CalculateReward(view, validatorSet)
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
func CalculateReward(view *st.StoreView, validatorSet *core.ValidatorSet) map[string]types.Coins {
	accountReward := map[string]types.Coins{}
	blockHeight := view.Height() + 1 // view points to the parent block
	if blockHeight < common.HeightEnableValidatorReward {
		grantValidatorsWithZeroReward(validatorSet, &accountReward)
	} else {
		grantStakerReward(view, validatorSet, &accountReward, blockHeight)
	} // TODO: calculate reward for the guardian nodes' stakers

	return accountReward
}

func grantValidatorsWithZeroReward(validatorSet *core.ValidatorSet, accountReward *map[string]types.Coins) {
	// Initial Mainnet release should not reward the validators until the guardians ready to deploy
	zeroReward := types.Coins{}.NoNil()
	for _, v := range validatorSet.Validators() {
		(*accountReward)[string(v.Address[:])] = zeroReward
	}
}

func grantStakerReward(view *st.StoreView, validatorSet *core.ValidatorSet, accountReward *map[string]types.Coins, blockHeight uint64) {
	isACheckpoint := blockHeight%uint64(checkpointInterval) == 0
	if !isACheckpoint {
		return
	}

	totalStake := validatorSet.TotalStake()
	if totalStake.Cmp(big.NewInt(0)) != 0 {

		stakeSourceMap := map[common.Address]*big.Int{}

		// TODO - Need to confirm: should we get the VCP from the current view? What if there is a stake deposit/withdraw?
		vcp := view.GetValidatorCandidatePool()
		for _, v := range validatorSet.Validators() {
			validatorAddr := v.Address
			stakeDelegate := vcp.FindStakeDelegate(validatorAddr)
			if stakeDelegate == nil { // should not happen
				panic(fmt.Sprintf("Failed to find stake delegate in the VCP: %v", hex.EncodeToString(validatorAddr[:])))
			}

			stakes := stakeDelegate.Stakes
			for _, stake := range stakes {
				if stake.Withdrawn {
					continue
				}
				stakeAmount := stake.Amount
				stakeSource := stake.Source
				if stakeAmountSum, exists := stakeSourceMap[stakeSource]; exists {
					stakeAmountSum := big.NewInt(0).Add(stakeAmountSum, stakeAmount)
					stakeSourceMap[stakeSource] = stakeAmountSum
				} else {
					stakeSourceMap[stakeSource] = stakeAmount
				}
			}
		}

		// the source of the stake divides the block reward proportional to their stake
		totalReward := big.NewInt(1).Mul(tfuelRewardPerBlock, big.NewInt(checkpointInterval))
		for stakeSourceAddr, stakeAmountSum := range stakeSourceMap {
			tmp := big.NewInt(1).Mul(totalReward, stakeAmountSum)
			rewardAmount := tmp.Div(tmp, totalStake)

			reward := types.Coins{
				ThetaWei: big.NewInt(0),
				TFuelWei: rewardAmount,
			}.NoNil()
			(*accountReward)[string(stakeSourceAddr[:])] = reward

			logger.Infof("Block reward for staker %v : %v", hex.EncodeToString(stakeSourceAddr[:]), reward)
		}
	}
}

func (exec *CoinbaseTxExecutor) getTxInfo(transaction types.Tx) *core.TxInfo {
	return &core.TxInfo{
		EffectiveGasPrice: exec.calculateEffectiveGasPrice(transaction),
	}
}

func (exec *CoinbaseTxExecutor) calculateEffectiveGasPrice(transaction types.Tx) *big.Int {
	return new(big.Int).SetUint64(0)
}
