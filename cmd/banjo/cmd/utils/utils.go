package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/bgentry/speakeasy"
	isatty "github.com/mattn/go-isatty"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/crypto"
	"github.com/thetatoken/ukulele/wallet"
	wtypes "github.com/thetatoken/ukulele/wallet/types"
)

var buf *bufio.Reader

func WalletUnlockAddress(cfgPath, addressStr string) (wtypes.Wallet, common.Address, *crypto.PublicKey, error) {
	wallet, err := wallet.OpenDefaultWallet(cfgPath)
	if err != nil {
		fmt.Printf("Failed to open wallet: %v\n", err)
		return nil, common.Address{}, nil, err
	}

	prompt := fmt.Sprintf("Please enter password: ")
	password, err := GetPassword(prompt)
	if err != nil {
		fmt.Printf("Failed to get password: %v\n", err)
		return nil, common.Address{}, nil, err
	}

	address := common.HexToAddress(addressStr)
	err = wallet.Unlock(address, password)
	if err != nil {
		fmt.Printf("Failed to unlock address %v: %v\n", address.Hex(), err)
		return nil, common.Address{}, nil, err
	}

	pubKey, err := wallet.GetPublicKey(address)
	if err != nil {
		fmt.Printf("Failed to get the public key for address %v: %v\n", address.Hex(), err)
		return nil, common.Address{}, nil, err
	}

	return wallet, address, pubKey, nil
}

func GetPassword(prompt string) (password string, err error) {
	if inputIsTty() {
		password, err = speakeasy.Ask(prompt)
	} else {
		password, err = stdinPassword()
	}
	return
}

func inputIsTty() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())
}

func stdinPassword() (string, error) {
	if buf == nil {
		buf = bufio.NewReader(os.Stdin)
	}
	password, err := buf.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(password), nil
}
