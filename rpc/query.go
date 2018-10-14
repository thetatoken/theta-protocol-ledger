package rpc

import (
	"encoding/hex"
	"errors"
	"net/http"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/ledger/types"
)

// ------------------------------- GetAccount -----------------------------------

type GetAccountArgs struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

type GetAccountResult struct {
	*types.Account
	Address string `json:"address"`
}

func (t *ThetaRPCServer) GetAccount(r *http.Request, args *GetAccountArgs, result *GetAccountResult) (err error) {
	if args.Address == "" {
		return errors.New("Address must be specified")
	}
	address := common.HexToAddress(args.Address)
	ledgerState, err := t.ledger.GetStateSnapshot()
	if err != nil {
		return err
	}
	result.Account = ledgerState.GetAccount(address)
	result.Address = args.Address
	return nil
}

// ------------------------------- GetSplitContract -----------------------------------

type GetSplitContractArgs struct {
	ResourceID string `json:"resource_id"`
}

type GetSplitContractResult struct {
	*types.SplitContract
}

func (t *ThetaRPCServer) GetSplitContract(r *http.Request, args *GetSplitContractArgs, result *GetSplitContractResult) (err error) {
	if args.ResourceID == "" {
		return errors.New("ResourceID must be specified")
	}
	resourceID, err := hex.DecodeString(args.ResourceID)
	if err != nil {
		return err
	}
	ledgerState, err := t.ledger.GetStateSnapshot()
	if err != nil {
		return err
	}
	result.SplitContract = ledgerState.GetSplitContract(resourceID)
	return nil
}
