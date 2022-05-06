package state

import (
	"bytes"
	"fmt"
	"math/big"

	log "github.com/sirupsen/logrus"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/crypto"
	"github.com/thetatoken/theta/ledger/types"
	"github.com/thetatoken/theta/rlp"
	"github.com/thetatoken/theta/store/database"
	"github.com/thetatoken/theta/store/treestore"
)

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "ledger"})

//
// ------------------------- StoreView -------------------------
//

type StoreView struct {
	height uint64 // block height
	store  *treestore.TreeStore

	coinbaseTransactinProcessed bool
	slashIntents                []types.SlashIntent
	refund                      uint64                 // Gas refund during smart contract execution
	logs                        []*types.Log           // Temporary store of events during smart contract execution
	balanceChanges              []*types.BalanceChange // Temporary store of balance changes during smart contract execution
}

// NewStoreView creates an instance of the StoreView
func NewStoreView(height uint64, root common.Hash, db database.Database) *StoreView {
	store := treestore.NewTreeStore(root, db)
	if store == nil {
		return nil
	}

	sv := &StoreView{
		height:       height,
		store:        store,
		slashIntents: []types.SlashIntent{},
		refund:       0,
	}
	return sv
}

// Copy returns a copy of the StoreView
func (sv *StoreView) Copy() (*StoreView, error) {
	copiedStore, err := sv.store.Copy()
	if err != nil {
		return nil, err
	}
	copiedStoreView := &StoreView{
		height:       sv.height,
		store:        copiedStore,
		slashIntents: []types.SlashIntent{},
		refund:       0,
	}
	return copiedStoreView, nil
}

// GetDB returns the underlying database.
func (sv *StoreView) GetDB() database.Database {
	return sv.store.GetDB()
}

// Hash returns the root hash of the tree store
func (sv *StoreView) Hash() common.Hash {
	return sv.store.Hash()
}

// Height returns the block height corresponding to the stored state
func (sv *StoreView) Height() uint64 {
	return sv.height
}

// IncrementHeight increments the block height by 1
func (sv *StoreView) IncrementHeight() {
	sv.height++
}

// Save saves the StoreView to the persistent storage, and return the root hash
func (sv *StoreView) Save() common.Hash {
	rootHash, err := sv.store.Commit()

	logger.Debugf("Commit to data store, height: %v, rootHash: %v", sv.height+1, rootHash.Hex())

	if err != nil {
		log.Panicf("Failed to save the StoreView: %v", err)
	}
	return rootHash
}

// Get returns the value corresponding to the key
func (sv *StoreView) Get(key common.Bytes) common.Bytes {
	value := sv.store.Get(key)
	return value
}

// Traverse traverses the trie and calls cb callback func on every key/value pair
// with key having prefix
func (sv *StoreView) Traverse(prefix common.Bytes, cb func(k, v common.Bytes) bool) bool {
	return sv.store.Traverse(prefix, cb)
}

func (sv *StoreView) ProveVCP(vcpKey []byte, vp *core.VCPProof) error {
	return sv.store.ProveVCP(vcpKey, vp)
}

// Delete removes the value corresponding to the key
func (sv *StoreView) Delete(key common.Bytes) {
	sv.store.Delete(key)
}

// Set returns the value corresponding to the key
func (sv *StoreView) Set(key common.Bytes, value common.Bytes) {
	sv.store.Set(key, value)
}

// AddSlashIntent adds slashIntent
func (sv *StoreView) AddSlashIntent(slashIntent types.SlashIntent) {
	sv.slashIntents = append(sv.slashIntents, slashIntent)
}

// GetSlashIntents retrieves all the slashIntents
func (sv *StoreView) GetSlashIntents() []types.SlashIntent {
	return sv.slashIntents
}

// ClearSlashIntents clears all the slashIntents
func (sv *StoreView) ClearSlashIntents() {
	sv.slashIntents = []types.SlashIntent{}
}

// CoinbaseTransactinProcessed returns whether the coinbase transaction for the current block has been processed
func (sv *StoreView) CoinbaseTransactinProcessed() bool {
	return sv.coinbaseTransactinProcessed
}

// SetCoinbaseTransactionProcessed sets whether the coinbase transaction for the current block has been processed
func (sv *StoreView) SetCoinbaseTransactionProcessed(processed bool) {
	sv.coinbaseTransactinProcessed = processed
}

// GetAccount returns an account.
func (sv *StoreView) GetAccount(addr common.Address) *types.Account {
	data := sv.Get(AccountKey(addr))
	if data == nil || len(data) == 0 {
		return nil
	}
	acc := &types.Account{}
	err := types.FromBytes(data, acc)
	if err != nil {
		log.Panicf("Error reading account %X error: %v",
			data, err.Error())
	}
	return acc
}

// // SetAccount sets an account.
// func (sv *StoreView) SetAccount(addr common.Address, acc *types.Account) {
// 	accBytes, err := types.ToBytes(acc)
// 	if err != nil {
// 		log.Panicf("Error writing account %v error: %v",
// 			acc, err.Error())
// 	}
// 	sv.Set(AccountKey(addr), accBytes)
// }

// SetAccount sets an account.
func (sv *StoreView) SetAccount(addr common.Address, acc *types.Account) {
	sv.setAccount(addr, acc, true)
}

func (sv *StoreView) setAccountWithoutStateTreeRefCountUpdate(addr common.Address, acc *types.Account) {
	sv.setAccount(addr, acc, false)
}

func (sv *StoreView) setAccount(addr common.Address, acc *types.Account, updateRefCountForAccountStateTree bool) {
	accBytes, err := types.ToBytes(acc)
	if err != nil {
		log.Panicf("Error writing account %v error: %v",
			acc, err.Error())
	}
	sv.Set(AccountKey(addr), accBytes)

	if !updateRefCountForAccountStateTree {
		return
	}

	if (acc == nil || acc.Root == common.Hash{}) || (acc.Root == core.EmptyRootHash) {
		return
	}

	tree := sv.getAccountStorage(acc)
	_, err = tree.Commit() // update the reference count of the account state trie root
	if err != nil {
		log.Panic(err)
	}
}

// DeleteAccount deletes an account.
func (sv *StoreView) DeleteAccount(addr common.Address) {
	sv.Delete(AccountKey(addr))
}

// SplitRuleExists checks if a split rule associated with the given resourceID already exists
func (sv *StoreView) SplitRuleExists(resourceID string) bool {
	return sv.GetSplitRule(resourceID) != nil
}

// AddSplitRule adds a split rule
func (sv *StoreView) AddSplitRule(splitRule *types.SplitRule) bool {
	if sv.SplitRuleExists(splitRule.ResourceID) {
		return false // Each resourceID can have at most one corresponding split rule
	}

	sv.SetSplitRule(splitRule.ResourceID, splitRule)
	return true
}

// UpdateSplitRule updates a split rule
func (sv *StoreView) UpdateSplitRule(splitRule *types.SplitRule) bool {
	if !sv.SplitRuleExists(splitRule.ResourceID) {
		return false
	}

	sv.SetSplitRule(splitRule.ResourceID, splitRule)
	return true
}

// GetSplitRule gets split rule.
func (sv *StoreView) GetSplitRule(resourceID string) *types.SplitRule {
	data := sv.Get(SplitRuleKey(resourceID))
	if data == nil || len(data) == 0 {
		return nil
	}
	splitRule := &types.SplitRule{}
	err := types.FromBytes(data, splitRule)
	if err != nil {
		log.Panicf("Error reading splitRule %X error: %v",
			data, err.Error())
	}
	return splitRule
}

// SetSplitRule sets split rule.
func (sv *StoreView) SetSplitRule(resourceID string, splitRule *types.SplitRule) {
	splitRuleBytes, err := types.ToBytes(splitRule)
	if err != nil {
		log.Panicf("Error writing splitRule %v error: %v",
			splitRule, err.Error())
	}
	sv.Set(SplitRuleKey(resourceID), splitRuleBytes)
}

// DeleteSplitRule deletes a split rule.
func (sv *StoreView) DeleteSplitRule(resourceID string) bool {
	key := SplitRuleKey(resourceID)
	deleted := sv.store.Delete(key)
	return deleted
}

// DeleteExpiredSplitRules deletes a split rule.
func (sv *StoreView) DeleteExpiredSplitRules(currentBlockHeight uint64) bool {
	prefix := SplitRuleKeyPrefix()

	expiredKeys := []common.Bytes{}
	sv.store.Traverse(prefix, func(key, value common.Bytes) bool {
		var splitRule types.SplitRule
		err := types.FromBytes(value, &splitRule)
		if err != nil {
			log.Panicf("Error reading splitRule %X error: %v", value, err.Error())
		}

		expired := (splitRule.EndBlockHeight < currentBlockHeight)
		if expired {
			expiredKeys = append(expiredKeys, key)
		}
		return true
	})

	for _, key := range expiredKeys {
		deleted := sv.store.Delete(key)
		if !deleted {
			logger.Errorf("Failed to delete expired split rules")
			return false
		}
	}

	return true
}

// GetValidatorCandidatePool gets the validator candidate pool.
func (sv *StoreView) GetValidatorCandidatePool() *core.ValidatorCandidatePool {
	data := sv.Get(ValidatorCandidatePoolKey())
	if data == nil || len(data) == 0 {
		return nil
	}
	vcp := &core.ValidatorCandidatePool{}
	err := types.FromBytes(data, vcp)
	if err != nil {
		log.Panicf("Error reading validator candidate pool %X, error: %v",
			data, err.Error())
	}
	return vcp
}

// UpdateValidatorCandidatePool updates the validator candidate pool.
func (sv *StoreView) UpdateValidatorCandidatePool(vcp *core.ValidatorCandidatePool) {
	vcpBytes, err := types.ToBytes(vcp)
	if err != nil {
		log.Panicf("Error writing validator candidate pool %v, error: %v",
			vcp, err.Error())
	}
	sv.Set(ValidatorCandidatePoolKey(), vcpBytes)
}

// GetGuardianCandidatePool gets the guardian candidate pool.
func (sv *StoreView) GetGuardianCandidatePool() *core.GuardianCandidatePool {
	data := sv.Get(GuardianCandidatePoolKey())
	if data == nil || len(data) == 0 {
		return core.NewGuardianCandidatePool()
	}
	gcp := &core.GuardianCandidatePool{}
	err := types.FromBytes(data, gcp)
	if err != nil {
		log.Panicf("Error reading guardian candidate pool %X, error: %v",
			data, err.Error())
	}
	return gcp
}

// UpdateGuardianCandidatePool updates the guardian candidate pool.
func (sv *StoreView) UpdateGuardianCandidatePool(gcp *core.GuardianCandidatePool) {
	gcpBytes, err := types.ToBytes(gcp)
	if err != nil {
		log.Panicf("Error writing guardian candidate pool %v, error: %v",
			gcp, err.Error())
	}
	sv.Set(GuardianCandidatePoolKey(), gcpBytes)
}

// GetStakeTransactionHeightList gets the heights of blocks that contain stake related transactions
func (sv *StoreView) GetStakeTransactionHeightList() *types.HeightList {
	data := sv.Get(StakeTransactionHeightListKey())
	if data == nil || len(data) == 0 {
		return nil
	}

	hl := &types.HeightList{}
	err := types.FromBytes(data, hl)
	if err != nil {
		log.Panicf("Error reading height list %X, error: %v",
			data, err.Error())
	}
	return hl
}

// UpdateStakeTransactionHeightList updates the heights of blocks that contain stake related transactions
func (sv *StoreView) UpdateStakeTransactionHeightList(hl *types.HeightList) {
	hlBytes, err := types.ToBytes(hl)
	if err != nil {
		log.Panicf("Error writing height list %v, error: %v",
			hl, err.Error())
	}
	sv.Set(StakeTransactionHeightListKey(), hlBytes)
}

type StakeWithHolder struct {
	Holder common.Address
	Stake  core.Stake
}

// GetEliteEdgeNodeStakeReturns gets the elite edge node stake returns
func (sv *StoreView) GetEliteEdgeNodeStakeReturns(height uint64) []StakeWithHolder {
	data := sv.Get(EliteEdgeNodeStakeReturnsKey(height))
	if data == nil || len(data) == 0 {
		return []StakeWithHolder{}
	}

	returnedStakes := []StakeWithHolder{}
	err := types.FromBytes(data, &returnedStakes)
	if err != nil {
		log.Panicf("Error reading elite edge stake returns %v, error: %v",
			data, err.Error())
	}
	return returnedStakes
}

// GetEliteEdgeNodeStakeReturns saves the elite edge node stake returns for the given height
func (sv *StoreView) SetEliteEdgeNodeStakeReturns(height uint64, stakeReturns []StakeWithHolder) {
	returnedStakesBytes, err := types.ToBytes(stakeReturns)
	if err != nil {
		log.Panicf("Error writing elite edge stake returns %v, error: %v",
			stakeReturns, err)
	}
	sv.Set(EliteEdgeNodeStakeReturnsKey(height), returnedStakesBytes)
}

// RemoveEliteEdgeNodeStakeReturns removes the elite edge node stake returns for the given height
func (sv *StoreView) RemoveEliteEdgeNodeStakeReturns(height uint64) {
	sv.Delete(EliteEdgeNodeStakeReturnsKey(height))
}

// GetTotalEENStake retrives the total active EEN stakes
func (sv *StoreView) GetTotalEENStake() *big.Int {
	raw := sv.Get(EliteEdgeNodesTotalActiveStakeKey())
	return new(big.Int).SetBytes(raw)
}

// SetTotalEENStake sets the total active EEN stakes
func (sv *StoreView) SetTotalEENStake(amount *big.Int) {
	sv.Set(EliteEdgeNodesTotalActiveStakeKey(), amount.Bytes())
}

func (sv *StoreView) GetStore() *treestore.TreeStore {
	return sv.store
}

func (sv *StoreView) ResetLogs() {
	sv.logs = []*types.Log{}
}

func (sv *StoreView) PopLogs() []*types.Log {
	ret := sv.logs
	sv.ResetLogs()
	return ret
}

func (sv *StoreView) ResetBalanceChanges() {
	sv.balanceChanges = []*types.BalanceChange{}
}

func (sv *StoreView) PopBalanceChanges() []*types.BalanceChange {
	ret := sv.balanceChanges
	sv.ResetBalanceChanges()
	return ret
}

//
// ---------- Implement vm.StateDB interface -----------
//

func (sv *StoreView) CreateAccount(addr common.Address) {
	account := types.NewAccount(addr)
	sv.SetAccount(addr, account)
}

func (sv *StoreView) GetOrCreateAccount(addr common.Address) *types.Account {
	account := sv.GetAccount(addr)
	if account != nil {
		return account
	}
	return types.NewAccount(addr)
}

func (sv *StoreView) CreateAccountWithPreviousBalance(addr common.Address) {
	account := types.NewAccount(addr)

	existingAccount := sv.GetAccount(addr)
	if existingAccount != nil { // only copy over the account balance, reset other fields including the account sequence
		account.Balance = existingAccount.Balance.NoNil()
	}

	sv.SetAccount(addr, account)
}

func (sv *StoreView) SubBalance(addr common.Address, amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	account := sv.GetAccount(addr)
	if account == nil {
		panic(fmt.Sprintf("Account for %v does not exist!", addr))
	}
	account.Balance = account.Balance.NoNil()
	account.Balance.TFuelWei.Sub(account.Balance.TFuelWei, amount)
	sv.SetAccount(addr, account)

	sv.addBalanceChange(&types.BalanceChange{
		Address:    addr,
		TokenType:  1,
		IsNegative: true,
		Delta:      new(big.Int).Set(amount),
	})
}

func (sv *StoreView) AddBalance(addr common.Address, amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	account := sv.GetOrCreateAccount(addr)
	account.Balance = account.Balance.NoNil()
	account.Balance.TFuelWei.Add(account.Balance.TFuelWei, amount)
	sv.SetAccount(addr, account)

	sv.addBalanceChange(&types.BalanceChange{
		Address:    addr,
		TokenType:  1,
		IsNegative: false,
		Delta:      new(big.Int).Set(amount),
	})
}

func (sv *StoreView) GetBalance(addr common.Address) *big.Int {
	return sv.GetOrCreateAccount(addr).Balance.TFuelWei
}

func (sv *StoreView) SubThetaBalance(addr common.Address, amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	account := sv.GetAccount(addr)
	if account == nil {
		panic(fmt.Sprintf("Account for %v does not exist!", addr))
	}
	account.Balance = account.Balance.NoNil()
	account.Balance.ThetaWei.Sub(account.Balance.ThetaWei, amount)
	sv.SetAccount(addr, account)

	sv.addBalanceChange(&types.BalanceChange{
		Address:    addr,
		TokenType:  0,
		IsNegative: true,
		Delta:      new(big.Int).Set(amount),
	})
}

func (sv *StoreView) AddThetaBalance(addr common.Address, amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	account := sv.GetOrCreateAccount(addr)
	account.Balance = account.Balance.NoNil()
	account.Balance.ThetaWei.Add(account.Balance.ThetaWei, amount)
	sv.SetAccount(addr, account)

	sv.addBalanceChange(&types.BalanceChange{
		Address:    addr,
		TokenType:  0,
		IsNegative: false,
		Delta:      new(big.Int).Set(amount),
	})
}

// GetThetaBalance returns the ThetaWei balance of the given address
func (sv *StoreView) GetThetaBalance(addr common.Address) *big.Int {
	return sv.GetOrCreateAccount(addr).Balance.ThetaWei
}

// GetThetaStake returns the total amount of ThetaWei the address staked to validators and/or guardians
func (sv *StoreView) GetThetaStake(addr common.Address) *big.Int {
	totalStake := big.NewInt(0)

	vcp := sv.GetValidatorCandidatePool()
	for _, v := range vcp.SortedCandidates {
		for _, stake := range v.Stakes {
			if stake.Source == addr {
				if stake.Withdrawn {
					continue // withdrawn stake does not count
				}
				totalStake = new(big.Int).Add(stake.Amount, totalStake)
			}
		}
	}

	gcp := sv.GetGuardianCandidatePool()
	for _, g := range gcp.SortedGuardians {
		for _, stake := range g.Stakes {
			if stake.Source == addr {
				if stake.Withdrawn {
					continue // withdrawn stake does not count
				}
				totalStake = new(big.Int).Add(stake.Amount, totalStake)
			}
		}
	}

	return totalStake
}

func (sv *StoreView) GetNonce(addr common.Address) uint64 {
	return sv.GetOrCreateAccount(addr).Sequence
}

func (sv *StoreView) SetNonce(addr common.Address, nonce uint64) {
	account := sv.GetOrCreateAccount(addr)
	account.Sequence = nonce
	sv.SetAccount(addr, account)
}

func (sv *StoreView) GetCodeHash(addr common.Address) common.Hash {
	account := sv.GetAccount(addr)
	if account == nil {
		return common.Hash{}
	}
	return account.CodeHash
}

func (sv *StoreView) GetCode(addr common.Address) []byte {
	account := sv.GetAccount(addr)
	if account == nil {
		return nil
	}
	if (account.CodeHash == types.EmptyCodeHash) || (account.CodeHash == core.SuicidedCodeHash) {
		return nil
	}
	return sv.GetCodeByHash(account.CodeHash)
}

func (sv *StoreView) GetCodeByHash(codeHash common.Hash) []byte {
	if codeHash == core.SuicidedCodeHash {
		return nil
	}
	codeKey := CodeKey(codeHash[:])
	return sv.Get(codeKey)
}

func (sv *StoreView) SetCode(addr common.Address, code []byte) {
	account := sv.GetOrCreateAccount(addr)
	codeHash := crypto.Keccak256Hash(code)
	account.CodeHash = codeHash
	sv.Set(CodeKey(account.CodeHash[:]), code)
	sv.SetAccount(addr, account)
}

func (sv *StoreView) GetCodeSize(addr common.Address) int {
	return len(sv.GetCode(addr))
}

func (sv *StoreView) AddRefund(gas uint64) {
	sv.refund += gas
}

func (sv *StoreView) SubRefund(gas uint64) {
	if gas > sv.refund {
		log.Panic("Refund counter below zero")
	}
	sv.refund -= gas
}

func (sv *StoreView) GetRefund() uint64 {
	return sv.refund
}

func (sv *StoreView) ResetRefund() {
	sv.refund = 0
}

func (sv *StoreView) GetBlockHeight() uint64 {
	blockHeight := sv.height + 1
	return blockHeight
}

func (sv *StoreView) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	return sv.GetState(addr, key)
}

func (sv *StoreView) getAccountStorage(account *types.Account) *treestore.TreeStore {
	return treestore.NewTreeStore(account.Root, sv.store.GetDB())
}

func (sv *StoreView) GetState(addr common.Address, key common.Hash) common.Hash {
	account := sv.GetAccount(addr)
	if account == nil {
		return common.Hash{}
	}
	logger.Debugf("StoreView.GetState, address: %v, account.root: %v, key: %v", addr, account.Root.Hex(), key.Hex())

	enc, err := sv.getAccountStorage(account).TryGet(key[:])
	if err != nil {
		log.Panic(err)
	}
	if len(enc) > 0 {
		_, content, _, err := rlp.Split(enc)
		if err != nil {
			log.Panic(err)
		}
		return common.BytesToHash(content)
	}
	return common.Hash{}
}

func (sv *StoreView) SetState(addr common.Address, key, val common.Hash) {
	account := sv.GetAccount(addr)
	if account == nil {
		account = types.NewAccount(addr)
	}
	tree := sv.getAccountStorage(account)
	if (val == common.Hash{}) {
		tree.TryDelete(key[:])
		root, err := tree.Commit()
		if err != nil {
			log.Panic(err)
		}
		account.Root = root
		sv.setAccountWithoutStateTreeRefCountUpdate(addr, account) // The ref counts of the state tree already got updated above
		logger.Debugf("StoreView.SetState, address: %v, account.root: %v, key: %v, val: %v", addr.Hex(), root.Hex(), key.Hex(), val.Hex())
		return
	}
	// Encoding []byte cannot fail, ok to ignore the error.
	v, _ := rlp.EncodeToBytes(bytes.TrimLeft(val[:], "\x00"))
	tree.TryUpdate(key[:], v)
	root, err := tree.Commit()
	if err != nil {
		log.Panic(err)
	}

	account.Root = root
	sv.setAccountWithoutStateTreeRefCountUpdate(addr, account) // The ref counts of the state tree already got updated above

	logger.Debugf("StoreView.SetState, address: %v, account.root: %v, key: %v, val: %v", addr.Hex(), root.Hex(), key.Hex(), val.Hex())
}

func (sv *StoreView) Suicide(addr common.Address) bool {
	account := sv.GetAccount(addr)
	if account == nil {
		return false
	}
	account.CodeHash = core.SuicidedCodeHash

	sv.addBalanceChange(&types.BalanceChange{
		Address:    addr,
		TokenType:  1,
		IsNegative: true,
		Delta:      new(big.Int).Set(account.Balance.TFuelWei),
	})

	account.Balance.TFuelWei = big.NewInt(0)
	sv.SetAccount(addr, account)
	return true
}

func (sv *StoreView) HasSuicided(addr common.Address) bool {
	account := sv.GetAccount(addr)
	if account == nil {
		return true
	}
	hasSuicided := (account.CodeHash == core.SuicidedCodeHash)
	return hasSuicided
}

// Exist reports whether the given account exists in state.
// Notably this should also return true for suicided accounts.
func (sv *StoreView) Exist(addr common.Address) bool {
	account := sv.GetAccount(addr)
	return account != nil
}

// Empty returns whether the given account is empty. Empty
// is defined according to EIP161 (balance = nonce = code = 0).
func (sv *StoreView) Empty(addr common.Address) bool {
	account := sv.GetAccount(addr)
	if account == nil {
		return true
	}
	return account.Sequence == 0 &&
		account.CodeHash == types.EmptyCodeHash &&
		account.Balance.IsZero()
}

func (sv *StoreView) RevertToSnapshot(root common.Hash) {
	var err error
	sv.store, err = sv.store.Revert(root) // revert to one of the previous roots
	if err != nil {
		log.Panic(err)
	}
}

func (sv *StoreView) Snapshot() common.Hash {
	sv.store.Trie.Commit(nil) // Needs to commit to the in-memory trie DB
	return sv.store.Hash()
}

func (sv *StoreView) Prune() error {
	err := sv.store.Prune(func(node []byte) bool {
		account := &types.Account{}
		err := types.FromBytes(node, account)
		if err != nil {
			return false
		}
		if (account.Root == (common.Hash{})) || (account.Root == core.EmptyRootHash) {
			return false
		}
		storage := sv.getAccountStorage(account)
		logger.Debugf("StoreView.Prune, address: %v, account.root: %v", account.Address, account.Root.Hex())

		err = storage.Prune(nil)
		if err != nil {
			logger.Errorf("Failed to prune storage for account %v", account)
			return false
		}
		return true
	})
	if err != nil {
		return fmt.Errorf("Failed to prune store view, %v", err)
	}
	return nil
}

func (sv *StoreView) AddLog(l *types.Log) {
	sv.logs = append(sv.logs, l)
}

func (sv *StoreView) addBalanceChange(bc *types.BalanceChange) {
	sv.balanceChanges = append(sv.balanceChanges, bc)
}
