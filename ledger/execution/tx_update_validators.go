package execution

import (
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/result"
	st "github.com/thetatoken/ukulele/ledger/state"
	"github.com/thetatoken/ukulele/ledger/types"
)

var _ TxExecutor = (*UpdateValidatorsTxExecutor)(nil)

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

func (exec *UpdateValidatorsTxExecutor) sanityCheck(chainID string, view *st.StoreView, transaction types.Tx) result.Result {
	// tx := transaction.(*types.UpdateValidatorsTx)

	// res := tx.Proposer.ValidateBasic()
	// if res.IsError() {
	// 	return res
	// }

	// // Get input account
	// proposerAccount, success := getInput(view, tx.Proposer)
	// if success.IsError() {
	// 	return result.Error("Proposer account does not exist: %v", tx.Proposer.Address)
	// }

	// // Validate input, advanced
	// signBytes := tx.SignBytes(chainID)
	// res = validateInputAdvanced(proposerAccount, signBytes, tx.Proposer)
	// if res.IsError() {
	// 	log.Infof(fmt.Sprintf("validateInputAdvanced failed on %X: %v", tx.Proposer.Address, res))
	// 	return res
	// }

	// if !sanityCheckForFee(tx.Fee) {
	//	return result.Error("Insufficient fee. Transaction fee needs to be at least %v GammaWei",
	//		types.MinimumTransactionFeeGammaWei).WithErrorCode(result.CodeInvalidFee)
	// }

	// // Verify that validator set matches with local config.
	// genDoc, err := ReadGenesisFile()
	// if err != nil {
	// 	return result.Error(err.Error())
	// }
	// configedValidators := make(map[string]ttypes.GenesisValidator)
	// for _, v := range genDoc.Validators {
	// 	configedValidators[hex.EncodeToString(v.PubKey.Bytes())] = v
	// }
	// for _, pv := range tx.Validators {
	// 	cv, ok := configedValidators[hex.EncodeToString(pv.PubKey)]
	// 	if !ok || cv.Amount != int64(pv.Power) {
	// 		return result.Error("Proposed validator set doesn't match with local configuration")
	// 	}
	// }

	return result.OK
}

func (exec *UpdateValidatorsTxExecutor) process(chainID string, view *st.StoreView, transaction types.Tx) (common.Hash, result.Result) {
	tx := transaction.(*types.UpdateValidatorsTx)

	// account, res := getInput(view, tx.Proposer)
	// if res.IsError() {
	// 	return nil, res
	// }

	// if !chargeFee(account, tx.Fee) {
	// 	return nil, result.Error("failed to charge transaction fee")
	// }

	// account.Sequence++
	// view.SetAccount(tx.Proposer.Address, account)

	// exec.state.SetValidatorDiff(tx.Validators)

	txHash := types.TxID(chainID, tx)
	return txHash, result.OK
}
