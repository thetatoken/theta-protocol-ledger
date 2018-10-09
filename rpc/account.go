package rpc

import (
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
