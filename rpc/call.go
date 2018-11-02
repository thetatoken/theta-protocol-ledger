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
	vmReturn        common.Bytes   `json:"vm_return"`
	contractAddress common.Address `json:"contract_address"`
	gasUsed         uint64         `json:"gas_used"`
	vmError         error          `json:"vm_error"`
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
	stateCopy, err := ledgerState.Copy()
	if err != nil {
		return err
	}
	vmRet, contractAddr, gasUsed, vmErr := vm.Execute(sctx, stateCopy)

	result.vmReturn = vmRet
	result.contractAddress = contractAddr
	result.gasUsed = gasUsed
	result.vmError = vmErr

	return nil
}
