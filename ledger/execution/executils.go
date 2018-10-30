package execution

import (
	"encoding/hex"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/result"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/crypto"
	"github.com/thetatoken/ukulele/ledger/state"
	"github.com/thetatoken/ukulele/ledger/types"
)

// --------------------------------- Execution Utilities -------------------------------------

// TODO: need to implement the following two functions
// // Read genesis file.
// func ReadGenesisFile() (genDoc *ttypes.GenesisDoc, err error) {
// 	cfg, err := tcmd.ParseConfig()
// 	if err != nil {
// 		return
// 	}

// 	return ReadGenesisFileByPath(cfg.GenesisFile())
// }

// func ReadGenesisFileByPath(path string) (genDoc *ttypes.GenesisDoc, err error) {
// 	genDocJSON, err := ioutil.ReadFile(path)
// 	if err != nil {
// 		return
// 	}

// 	genDoc, err = ttypes.GenesisDocFromJSON(genDocJSON)
// 	return
// }

// getValidatorAddresses returns validators' addresses
func getValidatorAddresses(consensus core.ConsensusEngine, valMgr core.ValidatorManager) []common.Address {
	epoch := consensus.GetEpoch()
	validators := valMgr.GetValidatorSetForEpoch(epoch).Validators()
	validatorAddresses := make([]common.Address, len(validators))
	for i, v := range validators {
		validatorAddresses[i] = v.Address()
	}
	return validatorAddresses
}

func isAValidator(pubKey *crypto.PublicKey, validatorAddresses []common.Address) result.Result {
	if pubKey == nil || pubKey.IsEmpty() {
		return result.Error("Null proposer pubKey detected in coinbaseTx sanity check")
	}
	addr := pubKey.Address()
	proposerIsAValidator := false
	for _, validatorAddr := range validatorAddresses {
		if addr == validatorAddr {
			proposerIsAValidator = true
			break
		}
	}
	if !proposerIsAValidator {
		return result.Error("The coinbaseTx proposer is not a validator")
	}

	return result.OK
}

// The accounts from the TxInputs must either already have
// crypto.PubKey.(type) != nil, (it must be known),
// or it must be specified in the TxInput.
func getInputs(view *state.StoreView, ins []types.TxInput) (map[string]*types.Account, result.Result) {
	accounts := map[string]*types.Account{}
	for _, in := range ins {
		// Account shouldn't be duplicated
		if _, ok := accounts[string(in.Address[:])]; ok {
			return nil, result.Error("getInputs - Duplicated address: %v", in.Address)
		}

		acc, success := getAccount(view, in.Address)
		if success.IsError() {
			return nil, result.Error("getInputs - Unknown address: %v", in.Address)
		}

		if in.PubKey != nil && !in.PubKey.IsEmpty() {
			acc.PubKey = in.PubKey
		}
		accounts[string(in.Address[:])] = acc
	}
	return accounts, result.OK
}

func getInput(view *state.StoreView, in types.TxInput) (*types.Account, result.Result) {
	return getOrMakeInputImpl(view, in, false)
}

func getOrMakeInput(view *state.StoreView, in types.TxInput) (*types.Account, result.Result) {
	return getOrMakeInputImpl(view, in, true)
}

// This function guarantees the public key of the retrieved account is not empty
func getOrMakeInputImpl(view *state.StoreView, in types.TxInput, makeNewAccount bool) (*types.Account, result.Result) {
	acc, success := getOrMakeAccountImpl(view, in.Address, makeNewAccount)
	if success.IsError() {
		return nil, result.Error("getOrMakeInputImpl - Unknown address: %v", in.Address)
	}

	// if in.Sequence == 1 && in.PubKey.Empty() {
	// 	return nil, result.Error("TxInput PubKey cannot be empty when Sequence == 1")
	// }

	if acc.PubKey == nil || acc.PubKey.IsEmpty() {
		acc.PubKey = in.PubKey
	}

	if acc.PubKey == nil || acc.PubKey.IsEmpty() {
		return nil, result.Error("TxInput PubKey cannot be nil or empty when Sequence == 1").WithErrorCode(result.CodeEmptyPubKeyWithSequence1)
	}

	return acc, result.OK
}

func getAccount(view *state.StoreView, address common.Address) (*types.Account, result.Result) {
	return getOrMakeAccountImpl(view, address, false)
}

func getOrMakeAccount(view *state.StoreView, address common.Address) *types.Account {
	acc, _ := getOrMakeAccountImpl(view, address, true)
	// Impossible to have error.
	return acc
}

func getOrMakeAccountImpl(view *state.StoreView, address common.Address, makeNewAccount bool) (*types.Account, result.Result) {
	acc := view.GetAccount(address)
	if acc == nil {
		if !makeNewAccount {
			return nil, result.Error("getOrMakeAccountImpl - Unknown address: %v", address)
		}
		acc = types.NewAccount()
		acc.LastUpdatedBlockHeight = view.Height()
	}
	acc.UpdateToHeight(view.Height())

	return acc, result.OK
}

func getOrMakeOutputs(view *state.StoreView, accounts map[string]*types.Account, outs []types.TxOutput) (map[string]*types.Account, result.Result) {
	if accounts == nil {
		accounts = make(map[string]*types.Account)
	}

	for _, out := range outs {
		// Account shouldn't be duplicated
		if _, ok := accounts[string(out.Address[:])]; ok {
			return nil, result.Error("getOrMakeOutputs - Duplicated address: %v", out.Address)
		}

		acc := getOrMakeAccount(view, out.Address)
		accounts[string(out.Address[:])] = acc
	}
	return accounts, result.OK
}

// Validate inputs basic structure
func validateInputsBasic(ins []types.TxInput) result.Result {
	for _, in := range ins {
		// Check TxInput basic
		if res := in.ValidateBasic(); res.IsError() {
			return res
		}
	}
	return result.OK
}

// Validate inputs and compute total amount of coins
func validateInputsAdvanced(accounts map[string]*types.Account, signBytes []byte, ins []types.TxInput) (total types.Coins, res result.Result) {
	total = types.NewCoins(0, 0)
	for _, in := range ins {
		acc := accounts[string(in.Address[:])]
		if acc == nil {
			panic("validateInputsAdvanced() expects account in accounts")
		}
		res = validateInputAdvanced(acc, signBytes, in)
		if res.IsError() {
			return
		}
		// Good. Add amount to total
		total = total.Plus(in.Coins)
	}
	return total, result.OK
}

func validateInputAdvanced(acc *types.Account, signBytes []byte, in types.TxInput) result.Result {
	// Check sequence/coins
	seq, balance := acc.Sequence, acc.Balance
	if seq+1 != in.Sequence {
		return result.Error("Got %v, expected %v. (acc.seq=%v)",
			in.Sequence, seq+1, acc.Sequence).WithErrorCode(result.CodeInvalidSequence)
	}

	// Check amount
	if !balance.IsGTE(in.Coins) {
		return result.Error("balance is %v, tried to send %v",
			balance, in.Coins).WithErrorCode(result.CodeInsufficientFund)
	}

	// Check pubkey
	if acc.PubKey.IsEmpty() {
		return result.Error("Account pubkey is nil!")
	}

	// Check signatures
	if !acc.PubKey.VerifySignature(signBytes, in.Signature) {
		return result.Error("Signature verification failed, SignBytes: %v",
			hex.EncodeToString(signBytes)).WithErrorCode(result.CodeInvalidSignature)
	}

	return result.OK
}

func validateOutputsBasic(outs []types.TxOutput) result.Result {
	for _, out := range outs {
		// Check TxOutput basic
		if res := out.ValidateBasic(); res.IsError() {
			return res
		}
	}
	return result.OK
}

func sumOutputs(outs []types.TxOutput) types.Coins {
	total := types.NewCoins(0, 0)
	for _, out := range outs {
		total = total.Plus(out.Coins)
	}
	return total
}

// Note: Since totalInput == totalOutput + fee, the transaction fee is charged implicitly
//       by the following adjustByInputs() function. No special handling needed
func adjustByInputs(view *state.StoreView, accounts map[string]*types.Account, ins []types.TxInput) {
	for _, in := range ins {
		acc := accounts[string(in.Address[:])]
		if acc == nil {
			panic("adjustByInputs() expects account in accounts")
		}
		if !acc.Balance.IsGTE(in.Coins) {
			panic("adjustByInputs() expects sufficient funds")
		}
		acc.Balance = acc.Balance.Minus(in.Coins)
		acc.Sequence++
		view.SetAccount(in.Address, acc)
	}
}

func adjustByOutputs(view *state.StoreView, accounts map[string]*types.Account, outs []types.TxOutput) {
	for _, out := range outs {
		acc := accounts[string(out.Address[:])]
		if acc == nil {
			panic("adjustByOutputs() expects account in accounts")
		}
		acc.Balance = acc.Balance.Plus(out.Coins)
		view.SetAccount(out.Address, acc)
	}
}

func sanityCheckForFee(fee types.Coins) bool {
	fee = fee.NoNil()
	return fee.ThetaWei.Cmp(types.Zero) == 0 && fee.GammaWei.Cmp(types.Zero) > 0
}

func chargeFee(account *types.Account, fee types.Coins) bool {
	if !account.Balance.IsGTE(fee) {
		return false
	}

	account.Balance = account.Balance.Minus(fee)
	return true
}
