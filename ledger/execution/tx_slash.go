package execution

import (
	"math/big"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/result"
	"github.com/thetatoken/theta/core"
	st "github.com/thetatoken/theta/ledger/state"
	"github.com/thetatoken/theta/ledger/types"
)

var _ TxExecutor = (*SlashTxExecutor)(nil)

// ------------------------------- Slash Transaction -----------------------------------

type SlashTxExecutor struct {
	consensus core.ConsensusEngine
	valMgr    core.ValidatorManager
}

// NewSlashTxExecutor creates a new instance of SlashTxExecutor
func NewSlashTxExecutor(consensus core.ConsensusEngine, valMgr core.ValidatorManager) *SlashTxExecutor {
	return &SlashTxExecutor{
		consensus: consensus,
		valMgr:    valMgr,
	}
}

func (exec *SlashTxExecutor) sanityCheck(chainID string, view *st.StoreView, viewSel core.ViewSelector, transaction types.Tx) result.Result {
	tx := transaction.(*types.SlashTx)

	validatorSet := getValidatorSet(exec.consensus.GetLedger(), exec.valMgr)
	validatorAddresses := getValidatorAddresses(validatorSet)

	// Validate proposer, basic
	res := tx.Proposer.ValidateBasic()
	if res.IsError() {
		return res
	}

	// verify the proposer is one of the validators
	res = isAValidator(tx.Proposer.Address, validatorAddresses)
	if res.IsError() {
		return res
	}

	proposerAccount, res := getInput(view, tx.Proposer)
	if res.IsError() {
		return res
	}

	// verify the proposer's signature
	signBytes := tx.SignBytes(chainID)
	if !tx.Proposer.Signature.Verify(signBytes, proposerAccount.Address) {
		return result.Error("SignBytes: %X", signBytes)
	}

	slashedAddress := tx.SlashedAddress
	slashedAccount := view.GetAccount(slashedAddress)
	if slashedAccount == nil {
		return result.Error("Account %v does not exist!", slashedAddress)
	}

	reservedFundFound := false
	for _, reservedFund := range slashedAccount.ReservedFunds {
		if reservedFund.ReserveSequence == tx.ReserveSequence {
			reservedFundFound = true
			break
		}
	}

	if !reservedFundFound {
		return result.Error("Reserved fund not found for %v", tx.ReserveSequence)
	}

	validatorAddress := tx.Proposer.Address
	validatorAccount := view.GetAccount(validatorAddress)
	if validatorAccount == nil {
		return result.Error("Validator %v does not exist!", validatorAddress)
	}

	overspendingProofBytes := tx.SlashProof
	slashProofVerified := exec.verifySlashProof(chainID, slashedAccount, overspendingProofBytes)
	if !slashProofVerified {
		return result.Error("Invalid slash proof: %v", overspendingProofBytes)
	}

	return result.OK
}

func (exec *SlashTxExecutor) process(chainID string, view *st.StoreView, viewSel core.ViewSelector, transaction types.Tx) (common.Hash, result.Result) {
	tx := transaction.(*types.SlashTx)

	slashedAddress := tx.SlashedAddress
	slashedAccount := view.GetAccount(slashedAddress)

	var reservedFundIdx int
	var reservedFund types.ReservedFund
	reservedFundFound := false
	for reservedFundIdx, reservedFund = range slashedAccount.ReservedFunds {
		if reservedFund.ReserveSequence == tx.ReserveSequence {
			reservedFundFound = true
			break
		}
	}

	if !reservedFundFound {
		return common.Hash{}, result.Error("Reserved fund not found for %v", tx.ReserveSequence)
	}

	proposerAddress := tx.Proposer.Address
	proposerAccount := view.GetAccount(proposerAddress)
	if proposerAccount == nil {
		return common.Hash{}, result.Error("Proposer %v does not exist!", proposerAddress)
	}

	// TODO: We should transfer the collateral to a special address, e.g. 0x0 instead of
	//       transferring to the proposer, so the proposer gain no extra benefit if it colludes with
	//       the address that overspent

	// Slash: transfer the collateral and remainding deposit to the validator that identified the overspending
	remainingFund := reservedFund.InitialFund.Minus(reservedFund.UsedFund)
	if !remainingFund.IsNonnegative() {
		remainingFund = types.NewCoins(0, 0) // Should NOT happen, just to be on the safe side
	}
	slashedAmount := reservedFund.Collateral.Plus(remainingFund)

	proposerAccount.Balance = proposerAccount.Balance.Plus(slashedAmount)
	slashedAccount.ReservedFunds = append(slashedAccount.ReservedFunds[:reservedFundIdx],
		slashedAccount.ReservedFunds[reservedFundIdx+1:]...)

	view.SetAccount(proposerAddress, proposerAccount)
	view.SetAccount(slashedAddress, slashedAccount)

	txHash := types.TxID(chainID, tx)
	return txHash, result.OK
}

func (exec *SlashTxExecutor) verifySlashProof(chainID string, slashedAccount *types.Account, overspendingProofBytes []byte) bool {
	var overspendingProof types.OverspendingProof
	err := types.FromBytes(overspendingProofBytes, &overspendingProof)
	if err != nil {
		// TODO: need proper logging and error handling here.
		//panic(fmt.Sprintf("Failed to parse overspending proof: %v\n", err))
		logger.Errorf("Failed to parse overspending proof: %v", err)
		return false
	}

	slashedAddress := slashedAccount.Address
	reserveSequence := overspendingProof.ReserveSequence
	for _, reservedFund := range slashedAccount.ReservedFunds {
		if reservedFund.ReserveSequence != reserveSequence {
			continue
		}

		settledPaymentLookup := make(map[string]bool)
		fundIntendedToSpend := types.NewCoins(0, 0)
		for _, servicePaymentTx := range overspendingProof.ServicePayments {
			if slashedAddress != servicePaymentTx.Source.Address {
				return false // servicePaymentTx does not come from the slashed account
			}

			if servicePaymentTx.ReserveSequence != overspendingProof.ReserveSequence {
				return false // servicePaymentTx does not belong to claimed reserved fund
			}

			sourceSignedBytes := servicePaymentTx.SourceSignBytes(chainID)
			if !servicePaymentTx.Source.Signature.Verify(sourceSignedBytes, slashedAccount.Address) {
				return false // servicePaymentTx not signed by the slashed account
			}

			paymentKey := string(servicePaymentTx.Target.Address[:]) + "." + string(servicePaymentTx.PaymentSequence)
			_, targetExists := settledPaymentLookup[paymentKey]
			if targetExists {
				return false // to prevent using partial payments as proof
			}
			settledPaymentLookup[paymentKey] = true

			fundIntendedToSpend = fundIntendedToSpend.Plus(servicePaymentTx.Source.Coins)
		}

		fundOverspent := !reservedFund.InitialFund.IsGTE(fundIntendedToSpend)
		return fundOverspent
	}

	return false
}

func (exec *SlashTxExecutor) getTxInfo(transaction types.Tx) *core.TxInfo {
	tx := transaction.(*types.SlashTx)
	return &core.TxInfo{
		Address:           tx.Proposer.Address,
		Sequence:          tx.Proposer.Sequence,
		EffectiveGasPrice: exec.calculateEffectiveGasPrice(transaction),
	}
}

func (exec *SlashTxExecutor) calculateEffectiveGasPrice(transaction types.Tx) *big.Int {
	return new(big.Int).SetUint64(0)
}
