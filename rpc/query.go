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

// ------------------------------- GetSplitRule -----------------------------------

type GetSplitRuleArgs struct {
	ResourceID string `json:"resource_id"`
}

type GetSplitRuleResult struct {
	*types.SplitRule
}

func (t *ThetaRPCServer) GetSplitRule(r *http.Request, args *GetSplitRuleArgs, result *GetSplitRuleResult) (err error) {
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
	result.SplitRule = ledgerState.GetSplitRule(resourceID)
	return nil
}
