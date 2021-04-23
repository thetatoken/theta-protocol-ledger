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
	"github.com/thetatoken/theta/common/util"
	"github.com/thetatoken/theta/core"
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
	guardianPool, eliteEdgeNodePool := RetrievePools(exec.chain, exec.db, tx.BlockHeight, guardianVotes, eliteEdgeNodeVotes)
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

func RetrievePools(chain *blockchain.Chain, db database.Database, blockHeight uint64, guardianVotes *core.AggregatedVotes,
	eliteEdgeNodeVotes *core.AggregatedEENVotes) (guardianPool *core.GuardianCandidatePool, eliteEdgeNodePool *core.EliteEdgeNodePool) {
	guardianPool = nil
	eliteEdgeNodePool = nil

	if blockHeight < common.HeightEnableTheta2 {
		guardianPool = nil
		eliteEdgeNodePool = nil
	} else if blockHeight < common.HeightEnableTheta3 {
		if guardianVotes != nil {
			guradianVoteBlock, err := chain.FindBlock(guardianVotes.Block)
			if err != nil {
				logger.Panic(err)
			}
			storeView := st.NewStoreView(guradianVoteBlock.Height, guradianVoteBlock.StateHash, db)
			guardianPool = storeView.GetGuardianCandidatePool()
		}
	} else { // blockHeight >= common.HeightEnableTheta3
		// won't reward the elite edge nodes without the guardian votes, since we need to guardian votes to confirm that
		// the edge nodes vote for the correct checkpoint
		if guardianVotes != nil {
			guradianVoteBlock, err := chain.FindBlock(guardianVotes.Block)
			if err != nil {
				logger.Panic(err)
			}
			storeView := st.NewStoreView(guradianVoteBlock.Height, guradianVoteBlock.StateHash, db)
			guardianPool = storeView.GetGuardianCandidatePool()

			if eliteEdgeNodeVotes != nil {
				if eliteEdgeNodeVotes.Block == guardianVotes.Block {
					eliteEdgeNodePool = storeView.GetEliteEdgeNodePoolOfLastCheckpoint()
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

	for addr, reward := range accountReward {
		logger.Infof("Total reward for staker %v : %v", hex.EncodeToString([]byte(addr)), reward)
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

	if blockHeight >= common.HeightEnableTheta3 {
		srdsr := view.GetStakeRewardDistributionRuleSet()
		handleGuardianNodeRewardSplit(accountReward, &stakeSourceMap, guardianPool, srdsr)
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

		if blockHeight >= common.HeightEnableTheta3 {
			srdsr := view.GetStakeRewardDistributionRuleSet()
			handleEliteEdgeNodeRewardSplit(accountReward, &stakeSourceMap, eliteEdgeNodePool, srdsr)
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

		logger.Infof("%v reward for staker %v : %v  (before split)", rewardType, hex.EncodeToString(stakeSourceAddr[:]), stakingReward)
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
		samples[i], err = rand.Int(util.NewHashRand(seed), totalStake)
		if err != nil {
			// Should not reach here
			logger.Panic(err)
		}

		// // ---------- Just for testing ---------- //
		// totalStakeFloat := new(big.Float).SetInt(totalStake)
		// sampleFloat := new(big.Float).SetInt(samples[i])
		// logger.Infof("RandSample -- r: %v, height: %v, totalStake: %v, sample[%v]: %v",
		// 	new(big.Float).Quo(sampleFloat, totalStakeFloat).Text('f', 6), view.Height()+1, totalStake, i, samples[i])
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

		logger.Infof("RandomReward -- staker: %v, count: %v, height: %v, stake: %v, type: %v", stakeSourceAddr, count, view.Height()+1, stakeAmountSum, rewardType)

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

			logger.Infof("%v reward for staker %v : %v (before split)", rewardType, hex.EncodeToString(stakeSourceAddr[:]), reward)
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

type BeneficiaryData struct {
	StakeAmount     *big.Int       // total amount of stake staked to the Holder
	Holder          common.Address // delegate address, i.e. the address of a gardian/elite edge node
	Beneficiary     common.Address // beneficiary for the reward split
	SplitBasisPoint uint           // An integer between 0 and 10000, representing the fraction of the reward the beneficiary should get (in terms of 1/10000), https://en.wikipedia.org/wiki/Basis_point
}

type SplitMetadata struct {
	StakeAmountSum      *big.Int // the total amount of stake from a staker, i.e. the "source" wallet
	BeneficiaryDataList []BeneficiaryData
}

func handleGuardianNodeRewardSplit(accountRewardMap *map[string]types.Coins, stakeAmountSumMap *map[common.Address]*big.Int,
	guardianPool *core.GuardianCandidatePool, srdrs *core.StakeRewardDistributionRuleSet) {
	splitMap := map[string](*SplitMetadata){}

	for _, gn := range guardianPool.SortedGuardians {
		stakeHolder := gn.Holder
		holderStakes := gn.Stakes
		addToSplitMap(stakeHolder, holderStakes, accountRewardMap, stakeAmountSumMap, srdrs, &splitMap)
	}

	handleRewardSplit(accountRewardMap, &splitMap)
}

func handleEliteEdgeNodeRewardSplit(accountRewardMap *map[string]types.Coins, stakeAmountSumMap *map[common.Address]*big.Int,
	eliteEdgeNodePool *core.EliteEdgeNodePool, srdrs *core.StakeRewardDistributionRuleSet) {
	splitMap := map[string](*SplitMetadata){}

	for _, een := range eliteEdgeNodePool.SortedEliteEdgeNodes {
		stakeHolder := een.Holder
		holderStakes := een.Stakes
		addToSplitMap(stakeHolder, holderStakes, accountRewardMap, stakeAmountSumMap, srdrs, &splitMap)
	}

	handleRewardSplit(accountRewardMap, &splitMap)
}

// splitMap: staker => staker's split metadata {staker's total stake, list of beneficiaries}
func addToSplitMap(stakeHolder common.Address, holderStakes []*core.Stake, accountRewardMap *map[string]types.Coins,
	stakeAmountSumMap *map[common.Address]*big.Int, srdrs *core.StakeRewardDistributionRuleSet,
	splitMap *(map[string](*SplitMetadata))) {
	rewardDistr := srdrs.GetWithStakeHolderAddress(stakeHolder)
	if rewardDistr == nil {
		return
	}

	if rewardDistr.StakeHolder != stakeHolder {
		logger.Panicf("Invalid reward distribution: rewardDistr.StakeHolder = %v, stakeHolder = %v",
			rewardDistr.StakeHolder, stakeHolder)
	}

	for _, stake := range holderStakes {
		var exists bool
		var stakeAmountSum *big.Int

		if stake.Withdrawn {
			continue
		}

		src := stake.Source
		if stakeAmountSum, exists = (*stakeAmountSumMap)[src]; !exists {
			continue
		}

		if _, exists = (*accountRewardMap)[string(src[:])]; !exists {
			continue
		}

		var splitMetadata *SplitMetadata
		if splitMetadata, exists = (*splitMap)[string(src[:])]; !exists {
			splitMetadata = &SplitMetadata{
				StakeAmountSum: stakeAmountSum,
			}
			(*splitMap)[string(src[:])] = splitMetadata
		}

		beneficiaryData := BeneficiaryData{
			StakeAmount:     stake.Amount,
			Holder:          rewardDistr.StakeHolder,
			Beneficiary:     rewardDistr.Beneficiary,
			SplitBasisPoint: rewardDistr.SplitBasisPoint,
		}

		splitMetadata.BeneficiaryDataList = append(splitMetadata.BeneficiaryDataList, beneficiaryData)
	}
}

func handleRewardSplit(accountRewardMap *map[string]types.Coins, splitMap *map[string](*SplitMetadata)) {
	srcAddrs := []string{}
	for srcAddr := range *accountRewardMap {
		srcAddrs = append(srcAddrs, srcAddr)
	}

	beneficiaryRewardMap := map[string]types.Coins{}
	for _, srcAddr := range srcAddrs {
		splitMetadata, exists := (*splitMap)[srcAddr]
		if !exists {
			continue
		}

		if splitMetadata.StakeAmountSum.Cmp(big.NewInt(0)) == 0 {
			continue
		}

		delegatedAmount := big.NewInt(0)
		for _, beneficiaryData := range splitMetadata.BeneficiaryDataList {
			delegatedAmount = new(big.Int).Add(delegatedAmount, beneficiaryData.StakeAmount)
		}
		if delegatedAmount.Cmp(splitMetadata.StakeAmountSum) > 0 { // should never happen
			logger.Panicf("Invalid split metadata: %v", splitMetadata)
		}

		reward := (*accountRewardMap)[srcAddr]
		for _, beneficiaryData := range splitMetadata.BeneficiaryDataList {
			// beneficiarySplit = reward.TFuelWei * (beneficiaryData.StakeAmount / StakeAmountSum) * (beneficiaryData.SplitBasisPoint / 10000)
			tmp := big.NewInt(1).Mul(reward.TFuelWei, beneficiaryData.StakeAmount)
			tmp = big.NewInt(1).Mul(tmp, big.NewInt(int64(beneficiaryData.SplitBasisPoint)))
			tmp = tmp.Div(tmp, splitMetadata.StakeAmountSum)
			beneficiarySplitAmount := tmp.Div(tmp, big.NewInt(10000))

			if beneficiarySplitAmount.Cmp(reward.TFuelWei) > 0 {
				logger.Panicf("Invalid split metadata: %v", splitMetadata)
			}

			reward.TFuelWei = new(big.Int).Sub(reward.TFuelWei, beneficiarySplitAmount)

			beneficiarySplitCoins := types.Coins{
				ThetaWei: big.NewInt(0),
				TFuelWei: beneficiarySplitAmount,
			}
			bAddr := string(beneficiaryData.Beneficiary[:])
			if br, ok := beneficiaryRewardMap[bAddr]; ok {
				beneficiaryRewardMap[bAddr] = br.Plus(beneficiarySplitCoins)
			} else {
				beneficiaryRewardMap[bAddr] = beneficiarySplitCoins
			}
		}

		(*accountRewardMap)[srcAddr] = reward
	}

	for bAddr, bReward := range beneficiaryRewardMap {
		if accReward, exists := (*accountRewardMap)[bAddr]; exists {
			(*accountRewardMap)[bAddr] = accReward.Plus(bReward)
		} else {
			(*accountRewardMap)[bAddr] = bReward
		}
	}
}
