package rpc

import (
	"github.com/thetatoken/theta/common"
)

// ------------------------------- CallSmartContract -----------------------------------

type CallSmartContractArgs struct {
	SctxBytes string `json:"sctx_bytes"`
}

type CallSmartContractResult struct {
	VmReturn        string            `json:"vm_return"`
	ContractAddress common.Address    `json:"contract_address"`
	GasUsed         common.JSONUint64 `json:"gas_used"`
	VmError         string            `json:"vm_error"`
}

// ---- Temporarily disable the CallSmartContract RPC API, more testing needed ---- //

// CallSmartContract calls the smart contract. However, calling a smart contract does NOT modify
// the globally consensus state. It can be used for dry run, or for retrieving info from smart contracts
// without actually spending gas.
func (t *ThetaRPCService) CallSmartContract(args *CallSmartContractArgs, result *CallSmartContractResult) (err error) {
	// sctxBytes, err := hex.DecodeString(args.SctxBytes)
	// if err != nil {
	// 	return err
	// }

	// tx, err := types.TxFromBytes(sctxBytes)
	// sctx, ok := tx.(*types.SmartContractTx)
	// if !ok {
	// 	return fmt.Errorf("Failed to parse SmartContractTx: %v", args.SctxBytes)
	// }

	// ledgerState, err := t.ledger.GetDeliveredSnapshot()
	// if err != nil {
	// 	return err
	// }
	// vmRet, contractAddr, gasUsed, vmErr := vm.Execute(sctx, ledgerState)
	// ledgerState.Save()

	// result.VmReturn = hex.EncodeToString(vmRet)
	// result.ContractAddress = contractAddr
	// result.GasUsed = common.JSONUint64(gasUsed)
	// if vmErr != nil {
	// 	result.VmError = vmErr.Error()
	// }

	return nil
}
