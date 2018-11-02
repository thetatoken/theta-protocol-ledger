package rpc

import (
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/ledger/vm"
)

// ------------------------------- CallSmartContract -----------------------------------

type CallSmartContractArgs struct {
	SctxBytes string `json:"sctx_bytes"`
}

type CallSmartContractResult struct {
	VmReturn        string         `json:"vm_return"`
	ContractAddress common.Address `json:"contract_address"`
	GasUsed         uint64         `json:"gas_used"`
	VmError         error          `json:"vm_error"`
}

// CallSmartContract calls the smart contract. However, calling a smart contract does NOT modify
// the globally consensus state. It can be used for dry run, or for retrieving info from smart contracts
// without actually spending gas.
func (t *ThetaRPCServer) CallSmartContract(r *http.Request, args *CallSmartContractArgs, result *CallSmartContractResult) (err error) {
	sctxBytes, err := hex.DecodeString(args.SctxBytes)
	if err != nil {
		return err
	}

	tx, err := types.TxFromBytes(sctxBytes)
	sctx, ok := tx.(*types.SmartContractTx)
	if !ok {
		return fmt.Errorf("Failed to parse SmartContractTx: %v", args.SctxBytes)
	}

	ledgerState, err := t.ledger.GetStateSnapshot()
	if err != nil {
		return err
	}
	vmRet, contractAddr, gasUsed, vmErr := vm.Execute(sctx, ledgerState)
	ledgerState.Save()

	result.VmReturn = hex.EncodeToString(vmRet)
	result.ContractAddress = contractAddr
	result.GasUsed = gasUsed
	result.VmError = vmErr

	return nil
}
