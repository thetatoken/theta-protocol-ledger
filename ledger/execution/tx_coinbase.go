package execution

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/thetatoken/theta/blockchain"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/result"
	"github.com/thetatoken/theta/core"
	st "github.com/thetatoken/theta/ledger/state"
	"github.com/thetatoken/theta/ledger/types"
	"github.com/thetatoken/theta/store/database"
)

var weiMultiplier = big.NewInt(1e18)
var tfuelRewardPerBlock = big.NewInt(1).Mul(big.NewInt(48), weiMultiplier)    // 48 TFUEL per block, corresponds to about 5% *initial* annual inflation rate. The inflation rate naturally approaches 0 as the chain grows.
var eenTfuelRewardPerBlock = big.NewInt(1).Mul(big.NewInt(38), weiMultiplier) // 38 TFUEL per block, corresponds to about 4% *initial* annual inflation rate. The inflation rate naturally approaches 0 as the chain grows.

var _ TxExecutor = (*CoinbaseTxExecutor)(nil)

// ------------------------------- Coinbase Transaction -----------------------------------

// CoinbaseTxExecutor implements the TxExecutor interface
type CoinbaseTxExecutor struct {
	db        database.Database
	chain     *blockchain.Chain
	state     *st.LedgerState
	consensus core.ConsensusEngine
	valMgr    core.ValidatorManager
}

// NewCoinbaseTxExecutor creates a new instance of CoinbaseTxExecutor
func NewCoinbaseTxExecutor(db database.Database, chain *blockchain.Chain, state *st.LedgerState, consensus core.ConsensusEngine, valMgr core.ValidatorManager) *CoinbaseTxExecutor {
	return &CoinbaseTxExecutor{
		db:        db,
		chain:     chain,
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
	var expectedRewards map[string]types.Coins
	currentBlock := exec.consensus.GetLedger().GetCurrentBlock()
	guardianVotes := currentBlock.GuardianVotes
	eliteEdgeNodeVotes := currentBlock.EliteEdgeNodeVotes

	//if tx.BlockHeight < common.HeightEnableTheta2 || guardianVotes == nil
	if tx.BlockHeight < common.HeightEnableTheta2 {
		expectedRewards = CalculateReward(view, validatorSet, nil, nil, nil, nil)
	} else if tx.BlockHeight < common.HeightEnableTheta3 {
		if guardianVotes != nil {
			guradianVoteBlock, err := exec.chain.FindBlock(guardianVotes.Block)
			if err != nil {
				logger.Panic(err)
			}
			storeView := st.NewStoreView(guradianVoteBlock.Height, guradianVoteBlock.StateHash, exec.db)
			guardianCandidatePool := storeView.GetGuardianCandidatePool()
			expectedRewards = CalculateReward(view, validatorSet, guardianVotes, guardianCandidatePool, nil, nil)
		} else {
			expectedRewards = CalculateReward(view, validatorSet, nil, nil, nil, nil)
		}
	} else { // tx.BlockHeight >= common.HeightEnableTheta3
		if guardianVotes != nil {
			guradianVoteBlock, err := exec.chain.FindBlock(guardianVotes.Block)
			if err != nil {
				logger.Panic(err)
			}
			storeView := st.NewStoreView(guradianVoteBlock.Height, guradianVoteBlock.StateHash, exec.db)
			guardianCandidatePool := storeView.GetGuardianCandidatePool()

			var eliteEdgeNodePool *core.EliteEdgeNodePool
			if eliteEdgeNodeVotes != nil {
				if eliteEdgeNodeVotes.Block == guardianVotes.Block {
					eliteEdgeNodePool = storeView.GetEliteEdgeNodePool()
				} else {
					logger.Warnf("Elite edge nodes vote for block %v, while guardians vote for block %v, skip rewarding the elite edge nodes",
						eliteEdgeNodeVotes.Block.Hex(), guardianVotes.Block.Hex())
				}
			} else {
				logger.Warnf("Elite edge nodes have no vote for block %v", guardianVotes.Block.Hex())
			}

			expectedRewards = CalculateReward(view, validatorSet, guardianVotes, guardianCandidatePool, eliteEdgeNodeVotes, eliteEdgeNodePool)
		} else {
			// won't reward the elite edge nodes without the guardian votes, since we need to guardian votes to confirm that
			// the edge nodes vote for the correct checkpoint
			expectedRewards = CalculateReward(view, validatorSet, nil, nil, nil, nil)
		}
	}

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
func CalculateReward(view *st.StoreView, validatorSet *core.ValidatorSet,
	guardianVotes *core.AggregatedVotes, guardianPool *core.GuardianCandidatePool,
	eliteEdgeNodeVotes *core.AggregatedEENVotes, eliteEdgeNodePool *core.EliteEdgeNodePool) map[string]types.Coins {
	accountReward := map[string]types.Coins{}
	blockHeight := view.Height() + 1 // view points to the parent block
	if blockHeight < common.HeightEnableValidatorReward {
		grantValidatorsWithZeroReward(validatorSet, &accountReward)
	} else if blockHeight < common.HeightEnableTheta2 {
		grantThetaStakerReward(view, validatorSet, nil, core.NewGuardianCandidatePool(), &accountReward, blockHeight)
	} else if blockHeight < common.HeightEnableTheta3 {
		grantThetaStakerReward(view, validatorSet, guardianVotes, guardianPool, &accountReward, blockHeight)
	} else { // blockHeight >= common.HeightEnableTheta3
		grantThetaStakerReward(view, validatorSet, guardianVotes, guardianPool, &accountReward, blockHeight)
		grantTFuelStakerReward(view, eliteEdgeNodeVotes, eliteEdgeNodePool, &accountReward, blockHeight)
	}

	return accountReward
}

func grantValidatorsWithZeroReward(validatorSet *core.ValidatorSet, accountReward *map[string]types.Coins) {
	// Initial Mainnet release should not reward the validators until the guardians ready to deploy
	zeroReward := types.Coins{}.NoNil()
	for _, v := range validatorSet.Validators() {
		(*accountReward)[string(v.Address[:])] = zeroReward
	}
}

func grantThetaStakerReward(view *st.StoreView, validatorSet *core.ValidatorSet, guardianVotes *core.AggregatedVotes,
	guardianPool *core.GuardianCandidatePool, accountReward *map[string]types.Coins, blockHeight uint64) {
	if !common.IsCheckPointHeight(blockHeight) {
		return
	}

	totalStake := validatorSet.TotalStake()

	if guardianPool != nil {
		guardianPool = guardianPool.WithStake()
	}

	if guardianPool != nil {
		for i, g := range guardianPool.SortedGuardians {
			if guardianVotes.Multiplies[i] == 0 {
				continue
			}
			totalStake.Add(totalStake, g.TotalStake())
		}
	}

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

		if guardianPool != nil {
			for i, g := range guardianPool.SortedGuardians {
				if guardianVotes.Multiplies[i] == 0 {
					continue
				}
				stakes := g.Stakes
				for _, stake := range stakes {
					if stake.Withdrawn {
						continue
					}
					stakeAmount := stake.Amount
					stakeSource := stake.Source
					if stakeAmountSum, exists := stakeSourceMap[stakeSource]; exists {
						stakeAmountSum.Add(stakeAmountSum, stakeAmount)
					} else {
						stakeSourceMap[stakeSource] = stakeAmount
					}
				}
			}
		}

		// the source of the stake divides the block reward proportional to their stake
		totalReward := big.NewInt(1).Mul(tfuelRewardPerBlock, big.NewInt(common.CheckpointInterval))
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

// TODO: random sampling
func grantTFuelStakerReward(view *st.StoreView, eliteEdgeNodeVotes *core.AggregatedEENVotes,
	eliteEdgeNodePool *core.EliteEdgeNodePool, accountReward *map[string]types.Coins, blockHeight uint64) {
	if !common.IsCheckPointHeight(blockHeight) {
		return
	}

	if eliteEdgeNodeVotes == nil || eliteEdgeNodePool == nil {
		return
	}

	totalStake := big.NewInt(0)

	if eliteEdgeNodePool != nil {
		eliteEdgeNodePool = eliteEdgeNodePool.WithStake()
	}

	if eliteEdgeNodePool != nil {
		for i, e := range eliteEdgeNodePool.SortedEliteEdgeNodes {
			if eliteEdgeNodeVotes.Multiplies[i] == 0 {
				continue
			}
			totalStake.Add(totalStake, e.TotalStake())
		}
	}

	if totalStake.Cmp(big.NewInt(0)) != 0 {
		stakeSourceMap := map[common.Address]*big.Int{}

		if eliteEdgeNodePool != nil {
			for i, e := range eliteEdgeNodePool.SortedEliteEdgeNodes {
				if eliteEdgeNodeVotes.Multiplies[i] == 0 {
					continue
				}
				stakes := e.Stakes
				for _, stake := range stakes {
					if stake.Withdrawn {
						continue
					}
					stakeAmount := stake.Amount
					stakeSource := stake.Source
					if stakeAmountSum, exists := stakeSourceMap[stakeSource]; exists {
						stakeAmountSum.Add(stakeAmountSum, stakeAmount)
					} else {
						stakeSourceMap[stakeSource] = stakeAmount
					}
				}
			}
		}

		// the source of the stake divides the block reward proportional to their stake
		totalReward := big.NewInt(1).Mul(eenTfuelRewardPerBlock, big.NewInt(common.CheckpointInterval))
		for stakeSourceAddr, stakeAmountSum := range stakeSourceMap {
			tmp := big.NewInt(1).Mul(totalReward, stakeAmountSum)
			rewardAmount := tmp.Div(tmp, totalStake)

			tfuelStakingReward := types.Coins{
				ThetaWei: big.NewInt(0),
				TFuelWei: rewardAmount,
			}.NoNil()

			logger.Infof("EEN reward for staker %v : %v", hex.EncodeToString(stakeSourceAddr[:]), tfuelStakingReward)

			staker := string(stakeSourceAddr[:])
			if thetaStakingReward, exists := (*accountReward)[staker]; exists {
				totalStakingReward := thetaStakingReward.NoNil().Plus(tfuelStakingReward)
				(*accountReward)[staker] = totalStakingReward
			} else {
				(*accountReward)[staker] = tfuelStakingReward
			}
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
