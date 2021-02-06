package execution

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"sort"

	"github.com/thetatoken/theta/blockchain"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/result"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/crypto"
	st "github.com/thetatoken/theta/ledger/state"
	"github.com/thetatoken/theta/ledger/types"
	"github.com/thetatoken/theta/store/database"
)

var weiMultiplier = big.NewInt(1e18)
var tfuelRewardPerBlock = big.NewInt(1).Mul(big.NewInt(48), weiMultiplier)    // 48 TFUEL per block, corresponds to about 5% *initial* annual inflation rate. The inflation rate naturally approaches 0 as the chain grows.
var eenTfuelRewardPerBlock = big.NewInt(1).Mul(big.NewInt(38), weiMultiplier) // 38 TFUEL per block, corresponds to about 4% *initial* annual inflation rate. The inflation rate naturally approaches 0 as the chain grows.
var tfuelRewardN = 400                                                        // Reward receiver sampling params

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
	guardianPool, eliteEdgeNodePool := exec.retrievePools(tx, guardianVotes, eliteEdgeNodeVotes)
	expectedRewards = CalculateReward(exec.consensus.GetLedger(), view, validatorSet, guardianVotes, guardianPool, eliteEdgeNodeVotes, eliteEdgeNodePool)

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

func (exec *CoinbaseTxExecutor) retrievePools(tx *types.CoinbaseTx, guardianVotes *core.AggregatedVotes,
	eliteEdgeNodeVotes *core.AggregatedEENVotes) (guardianPool *core.GuardianCandidatePool, eliteEdgeNodePool *core.EliteEdgeNodePool) {
	guardianPool = nil
	eliteEdgeNodePool = nil

	if tx.BlockHeight < common.HeightEnableTheta2 {
		guardianPool = nil
		eliteEdgeNodePool = nil
	} else if tx.BlockHeight < common.HeightEnableTheta3 {
		if guardianVotes != nil {
			guradianVoteBlock, err := exec.chain.FindBlock(guardianVotes.Block)
			if err != nil {
				logger.Panic(err)
			}
			storeView := st.NewStoreView(guradianVoteBlock.Height, guradianVoteBlock.StateHash, exec.db)
			guardianPool = storeView.GetGuardianCandidatePool()
		}
	} else { // tx.BlockHeight >= common.HeightEnableTheta3
		// won't reward the elite edge nodes without the guardian votes, since we need to guardian votes to confirm that
		// the edge nodes vote for the correct checkpoint
		if guardianVotes != nil {
			guradianVoteBlock, err := exec.chain.FindBlock(guardianVotes.Block)
			if err != nil {
				logger.Panic(err)
			}
			storeView := st.NewStoreView(guradianVoteBlock.Height, guradianVoteBlock.StateHash, exec.db)
			guardianPool = storeView.GetGuardianCandidatePool()

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
		}
	}

	return guardianPool, eliteEdgeNodePool
}

// CalculateReward calculates the block reward for each account
func CalculateReward(ledger core.Ledger, view *st.StoreView, validatorSet *core.ValidatorSet,
	guardianVotes *core.AggregatedVotes, guardianPool *core.GuardianCandidatePool,
	eliteEdgeNodeVotes *core.AggregatedEENVotes, eliteEdgeNodePool *core.EliteEdgeNodePool) map[string]types.Coins {
	accountReward := map[string]types.Coins{}
	blockHeight := view.Height() + 1 // view points to the parent block
	if blockHeight < common.HeightEnableValidatorReward {
		grantValidatorsWithZeroReward(validatorSet, &accountReward)
	} else if blockHeight < common.HeightEnableTheta2 || guardianVotes == nil || guardianPool == nil {
		grantValidatorReward(ledger, view, validatorSet, &accountReward, blockHeight)
	} else if blockHeight < common.HeightEnableTheta3 {
		grantValidatorAndGuardianReward(ledger, view, validatorSet, guardianVotes, guardianPool, &accountReward, blockHeight)
	} else { // blockHeight >= common.HeightEnableTheta3
		grantValidatorAndGuardianReward(ledger, view, validatorSet, guardianVotes, guardianPool, &accountReward, blockHeight)
		grantEliteEdgeNodeReward(ledger, view, guardianVotes, eliteEdgeNodeVotes, eliteEdgeNodePool, &accountReward, blockHeight)
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

func grantValidatorReward(ledger core.Ledger, view *st.StoreView, validatorSet *core.ValidatorSet, accountReward *map[string]types.Coins, blockHeight uint64) {
	if !common.IsCheckPointHeight(blockHeight) {
		return
	}

	totalStake := validatorSet.TotalStake()

	if totalStake.Cmp(big.NewInt(0)) == 0 {
		// Should never happen
		return
	}

	stakeSourceMap := map[common.Address]*big.Int{}
	stakeSourceList := []common.Address{}

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
				stakeSourceList = append(stakeSourceList, stakeSource)
			}
		}
	}

	totalReward := big.NewInt(1).Mul(tfuelRewardPerBlock, big.NewInt(common.CheckpointInterval))

	// the source of the stake divides the block reward proportional to their stake
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

// grant block rewards to both the validators and active guardians (they are both theta stakers)
func grantValidatorAndGuardianReward(ledger core.Ledger, view *st.StoreView, validatorSet *core.ValidatorSet, guardianVotes *core.AggregatedVotes,
	guardianPool *core.GuardianCandidatePool, accountReward *map[string]types.Coins, blockHeight uint64) {
	if !common.IsCheckPointHeight(blockHeight) {
		return
	}

	totalStake := validatorSet.TotalStake()

	if guardianPool == nil || guardianVotes == nil {
		// Should never reach here
		panic("guardianPool == nil || guardianVotes == nil")
	}
	guardianPool = guardianPool.WithStake()

	if totalStake.Cmp(big.NewInt(0)) == 0 {
		// Should never happen
		return
	}

	stakeSourceMap := map[common.Address]*big.Int{}
	stakeSourceList := []common.Address{}

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
				stakeSourceList = append(stakeSourceList, stakeSource)
			}
		}
	}

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

			totalStake.Add(totalStake, stakeAmount)

			if stakeAmountSum, exists := stakeSourceMap[stakeSource]; exists {
				stakeAmountSum.Add(stakeAmountSum, stakeAmount)
			} else {
				stakeSourceMap[stakeSource] = stakeAmount
				stakeSourceList = append(stakeSourceList, stakeSource)
			}
		}
	}

	totalReward := big.NewInt(1).Mul(tfuelRewardPerBlock, big.NewInt(common.CheckpointInterval))

	if blockHeight < common.HeightSampleStakingReward {
		// the source of the stake divides the block reward proportional to their stake
		issueFixedReward(stakeSourceMap, totalStake, accountReward, totalReward, "Block")
	} else {
		// randomly select (proportional to the stake) a constant-sized set of stakers and grand the block reward
		issueRandomizedReward(ledger, guardianVotes, view, stakeSourceList, stakeSourceMap,
			totalStake, accountReward, totalReward, "Block")
	}
}

// grant uptime mining rewards to active elite edge nodes (they are the tfuel stakers)
func grantEliteEdgeNodeReward(ledger core.Ledger, view *st.StoreView, guardianVotes *core.AggregatedVotes, eliteEdgeNodeVotes *core.AggregatedEENVotes,
	eliteEdgeNodePool *core.EliteEdgeNodePool, accountReward *map[string]types.Coins, blockHeight uint64) {
	if !common.IsCheckPointHeight(blockHeight) {
		return
	}

	if guardianVotes == nil {
		// Should never reach here
		panic("guardianVotes == nil")
	}

	if eliteEdgeNodeVotes == nil || eliteEdgeNodePool == nil {
		return
	}

	totalStake := big.NewInt(0)
	eliteEdgeNodePool = eliteEdgeNodePool.WithStake()
	for i, e := range eliteEdgeNodePool.SortedEliteEdgeNodes {
		if eliteEdgeNodeVotes.Multiplies[i] == 0 {
			continue
		}
		totalStake.Add(totalStake, e.TotalStake())
	}

	if totalStake.Cmp(big.NewInt(0)) != 0 {
		stakeSourceMap := map[common.Address]*big.Int{}
		stakeSourceList := []common.Address{}

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
					stakeSourceList = append(stakeSourceList, stakeSource)
				}
			}
		}

		// the source of the stake divides the block reward proportional to their stake
		totalReward := big.NewInt(1).Mul(eenTfuelRewardPerBlock, big.NewInt(common.CheckpointInterval))
		if blockHeight < common.HeightSampleStakingReward {
			// the source of the stake divides the block reward proportional to their stake
			issueFixedReward(stakeSourceMap, totalStake, accountReward, totalReward, "EEN  ")
		} else {
			// randomly select (proportional to the stake) a constant-sized set of stakers and grand the block reward
			issueRandomizedReward(ledger, guardianVotes, view, stakeSourceList, stakeSourceMap,
				totalStake, accountReward, totalReward, "EEN  ")
		}
	}
}

func issueFixedReward(stakeSourceMap map[common.Address]*big.Int, totalStake *big.Int, accountReward *map[string]types.Coins, totalReward *big.Int, rewardType string) {
	for stakeSourceAddr, stakeAmountSum := range stakeSourceMap {
		tmp := big.NewInt(1).Mul(totalReward, stakeAmountSum)
		rewardAmount := tmp.Div(tmp, totalStake)

		stakingReward := types.Coins{
			ThetaWei: big.NewInt(0),
			TFuelWei: rewardAmount,
		}.NoNil()

		staker := string(stakeSourceAddr[:])
		if existingReward, exists := (*accountReward)[staker]; exists {
			totalStakingReward := existingReward.NoNil().Plus(stakingReward)
			(*accountReward)[staker] = totalStakingReward
		} else {
			(*accountReward)[staker] = stakingReward
		}

		logger.Infof("%v reward for staker %v : %v", rewardType, hex.EncodeToString(stakeSourceAddr[:]), stakingReward)
	}
}

func issueRandomizedReward(ledger core.Ledger, guardianVotes *core.AggregatedVotes, view *st.StoreView, stakeSourceList []common.Address, stakeSourceMap map[common.Address]*big.Int,
	totalStake *big.Int, accountReward *map[string]types.Coins, totalReward *big.Int, rewardType string) {

	if guardianVotes == nil {
		// Should never reach here
		panic("guardianVotes == nil")
	}

	samples := make([]*big.Int, tfuelRewardN)
	for i := 0; i < tfuelRewardN; i++ {
		// Set random seed to (block_height||sampling_index||checkpoint_hash)
		seed := make([]byte, 2*binary.MaxVarintLen64+common.HashLength)
		binary.PutUvarint(seed[:], view.Height())
		binary.PutUvarint(seed[binary.MaxVarintLen64:], uint64(i))
		copy(seed[2*binary.MaxVarintLen64:], guardianVotes.Block[:])

		var err error
		samples[i], err = rand.Int(NewHashRand(seed), totalStake)
		if err != nil {
			// Should not reach here
			logger.Panic(err)
		}
	}

	sort.Sort(BigIntSort(samples))

	curr := 0
	currSum := big.NewInt(0)

	for i := 0; i < len(stakeSourceList); i++ {
		stakeSourceAddr := stakeSourceList[i]
		stakeAmountSum := stakeSourceMap[stakeSourceAddr]

		if curr >= tfuelRewardN {
			break
		}

		count := 0
		lower := currSum
		upper := new(big.Int).Add(currSum, stakeAmountSum)
		for curr < tfuelRewardN && samples[curr].Cmp(lower) >= 0 && samples[curr].Cmp(upper) < 0 {
			count++
			curr++
		}
		currSum = upper

		if count > 0 {
			tmp := new(big.Int).Mul(totalReward, big.NewInt(int64(count)))
			rewardAmount := tmp.Div(tmp, big.NewInt(int64(tfuelRewardN)))

			reward := types.Coins{
				ThetaWei: big.NewInt(0),
				TFuelWei: rewardAmount,
			}.NoNil()

			staker := string(stakeSourceAddr[:])
			if thetaStakingReward, exists := (*accountReward)[staker]; exists {
				totalStakingReward := thetaStakingReward.NoNil().Plus(reward)
				(*accountReward)[staker] = totalStakingReward
			} else {
				(*accountReward)[staker] = reward
			}

			logger.Infof("%v reward for staker %v : %v", rewardType, hex.EncodeToString(stakeSourceAddr[:]), reward)
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

type BigIntSort []*big.Int

func (s BigIntSort) Len() int           { return len(s) }
func (s BigIntSort) Less(i, j int) bool { return s[i].Cmp(s[j]) < 0 }
func (s BigIntSort) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// HashRand generate infinite number of random bytes by repeatedly hashing the seed
type HashRand struct {
	remaining []byte
	curr      common.Hash
}

func NewHashRand(seed []byte) *HashRand {
	return &HashRand{
		remaining: []byte{},
		curr:      crypto.Keccak256Hash(seed),
	}
}

func (r *HashRand) Read(buf []byte) (int, error) {
	if len(r.remaining) != 0 {
		n := copy(buf, r.remaining)
		r.remaining = r.remaining[n:]
		return n, nil
	}
	r.curr = crypto.Keccak256Hash(r.curr[:])
	n := copy(buf, r.curr[:])
	r.remaining = r.curr[n:]
	return n, nil
}
