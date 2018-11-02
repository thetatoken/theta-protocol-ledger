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

// ------------------------------ GetTransaction -----------------------------------

type GetTransactionArgs struct {
	Hash string `json:"hash"`
}

type GetTransactionResult struct {
	BlockHash   common.Hash `json:"block_hash"`
	BlockHeight uint64      `json:"block_height"`
	Status      TxStatus    `json:"status"`
	TxHash      common.Hash `json:"hash"`
	Tx          types.Tx    `json:"transaction"`
}

type TxStatus string

const (
	TxStatusNotFound  = "not_found"
	TxStatusPending   = "pending"
	TxStatusFinalized = "finalized"
)

func (t *ThetaRPCServer) GetTransaction(r *http.Request, args *GetTransactionArgs, result *GetTransactionResult) (err error) {
	if args.Hash == "" {
		return errors.New("Transanction hash must be specified")
	}
	hash := common.HexToHash(args.Hash)
	raw, block, found := t.chain.FindTxByHash(hash)
	if !found {
		result.Status = TxStatusNotFound
		return nil
	}
	result.TxHash = hash
	result.BlockHash = common.BytesToHash(block.Hash)
	result.BlockHeight = block.Height

	if block.Finalized {
		result.Status = TxStatusFinalized
	} else {
		result.Status = TxStatusPending
	}

	tx, err := types.TxFromBytes(raw)
	if err != nil {
		return err
	}
	result.Tx = tx

	return nil
}
