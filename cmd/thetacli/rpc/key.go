package rpc

import (
	"github.com/thetatoken/theta/common"
)

// ------------------------------- UnlockKey -----------------------------------

type UnlockKeyArgs struct {
	Address  string `json:"address"`
	Password string `json:"password"`
}

type UnlockKeyResult struct {
}

func (t *ThetaCliRPCService) UnlockKey(args *UnlockKeyArgs, result *UnlockKeyResult) (err error) {
	address := common.HexToAddress(args.Address)
	password := args.Password
	err = t.wallet.Unlock(address, password)
	return err
}

// ------------------------------- LockKey -----------------------------------

type LockKeyArgs struct {
	Address string `json:"address"`
}

type LockKeyResult struct {
}

func (t *ThetaCliRPCService) Lock(args *LockKeyArgs, result *LockKeyResult) (err error) {
	address := common.HexToAddress(args.Address)
	err = t.wallet.Lock(address)
	return err
}

// ------------------------------- NewKey -----------------------------------

type NewKeyArgs struct {
	Password string `json:"password"`
}

type NewKeyResult struct {
	Address string `json:"address"`
}

func (t *ThetaCliRPCService) NewKey(args *NewKeyArgs, result *NewKeyResult) (err error) {
	password := args.Password

	address, err := t.wallet.NewKey(password)
	if err != nil {
		return err
	}

	result.Address = address.Hex()
	return nil
}

// ------------------------------- ListKeys -----------------------------------

type ListKeysArgs struct {
}

type ListKeysResult struct {
	Addresses []string `json:"addresses"`
}

func (t *ThetaCliRPCService) ListKeys(args *ListKeysArgs, result *ListKeysResult) (err error) {
	addresses, err := t.wallet.List()
	if err != nil {
		return err
	}

	for _, address := range addresses {
		result.Addresses = append(result.Addresses, address.Hex())
	}

	return nil
}
