package execution

import (
	st "github.com/thetatoken/ukulele/ledger/state"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/ledger/types/result"
)

// ------------------------------- UpdateValidators Transaction -----------------------------------

// UpdateValidatorsTxExecutor implements the TxExecutor interface
type UpdateValidatorsTxExecutor struct {
	state *st.LedgerState
}

// NewUpdateValidatorsTxExecutor creates a new instance of UpdateValidatorsTxExecutor
func NewUpdateValidatorsTxExecutor(state *st.LedgerState) *UpdateValidatorsTxExecutor {
	return &UpdateValidatorsTxExecutor{
		state: state,
	}
}

func (exec *UpdateValidatorsTxExecutor) sanityCheck(chainID string, view types.ViewDataGetter, transaction types.Tx) result.Result {
	// tx := transaction.(*types.UpdateValidatorsTx)

	// res := tx.Proposer.ValidateBasic()
	// if res.IsErr() {
	// 	return res
	// }

	// // Get input account
	// proposerAccount, success := getInput(view, tx.Proposer)
	// if success.IsErr() {
	// 	return result.ErrBaseUnknownAddress.AppendLog(fmt.Sprintf("Proposer account does not exist: %v", tx.Proposer.Address))
	// }

	// // Validate input, advanced
	// signBytes := tx.SignBytes(chainID)
	// res = validateInputAdvanced(proposerAccount, signBytes, tx.Proposer)
	// if res.IsErr() {
	// 	log.Infof(fmt.Sprintf("validateInputAdvanced failed on %X: %v", tx.Proposer.Address, res))
	// 	return res.PrependLog("in validateInputAdvanced()")
	// }

	// if !sanityCheckForFee(tx.Fee) {
	// 	return result.ErrInternalError.PrependLog("invalid fee")
	// }

	// // Verify that validator set matches with local config.
	// genDoc, err := ReadGenesisFile()
	// if err != nil {
	// 	return result.ErrInternalError.PrependLog(err.Error())
	// }
	// configedValidators := make(map[string]ttypes.GenesisValidator)
	// for _, v := range genDoc.Validators {
	// 	configedValidators[hex.EncodeToString(v.PubKey.Bytes())] = v
	// }
	// for _, pv := range tx.Validators {
	// 	cv, ok := configedValidators[hex.EncodeToString(pv.PubKey)]
	// 	if !ok || cv.Amount != int64(pv.Power) {
	// 		return result.ErrInternalError.PrependLog("Proposed validator set doesn't match with local configuration")
	// 	}
	// }

	return result.OK
}

func (exec *UpdateValidatorsTxExecutor) process(chainID string, view types.ViewDataAccessor, transaction types.Tx) result.Result {
	// tx := transaction.(*types.UpdateValidatorsTx)

	// account, res := getInput(view, tx.Proposer)
	// if res.IsErr() {
	// 	return res.PrependLog("in getInput()")
	// }

	// if !chargeFee(account, tx.Fee) {
	// 	return result.ErrInternalError.AppendLog("failed to charge transaction fee")
	// }

	// account.Sequence++
	// view.SetAccount(tx.Proposer.Address, account)

	// exec.state.SetValidatorDiff(tx.Validators)

	return result.OK
}
