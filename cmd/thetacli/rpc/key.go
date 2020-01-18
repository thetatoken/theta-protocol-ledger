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
	Unlocked bool `json:"unlocked"`
}

func (t *ThetaCliRPCService) UnlockKey(args *UnlockKeyArgs, result *UnlockKeyResult) (err error) {
	address := common.HexToAddress(args.Address)
	password := args.Password
	err = t.wallet.Unlock(address, password, nil)
	if err != nil {
		result.Unlocked = false
		return err
	}
	result.Unlocked = t.wallet.IsUnlocked(address)
	return nil
}

// ------------------------------- LockKey -----------------------------------

type LockKeyArgs struct {
	Address string `json:"address"`
}

type LockKeyResult struct {
	Unlocked bool `json:"unlocked"`
}

func (t *ThetaCliRPCService) LockKey(args *LockKeyArgs, result *LockKeyResult) (err error) {
	address := common.HexToAddress(args.Address)
	err = t.wallet.Lock(address)
	result.Unlocked = t.wallet.IsUnlocked(address)
	if err != nil {
		return err
	}
	return nil
}

// ------------------------------- IsKeyUnlocked -----------------------------------

type IsKeyUnlockedArgs struct {
	Address string `json:"address"`
}

type IsKeyUnlockedResult struct {
	Unlocked bool `json:"unlocked"`
}

func (t *ThetaCliRPCService) IsKeyUnlocked(args *IsKeyUnlockedArgs, result *IsKeyUnlockedResult) (err error) {
	address := common.HexToAddress(args.Address)
	isKeyUnlocked := t.wallet.IsUnlocked(address)
	result.Unlocked = isKeyUnlocked
	return nil
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
