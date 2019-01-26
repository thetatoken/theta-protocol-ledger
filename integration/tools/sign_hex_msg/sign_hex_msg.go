package main

import (
	"encoding/hex"
	"flag"
	"fmt"

	"github.com/thetatoken/theta/cmd/thetacli/cmd/utils"
	"github.com/thetatoken/theta/common"
	ks "github.com/thetatoken/theta/wallet/softwallet/keystore"
)

//
// Usage:   sign_hex_msg -signer=<signer_address> -keys_dir=<keys_dir> -msg=<hex_msg_to_be_signed> -encrypted=<true/false>
//
// Example: sign_hex_msg -signer=2E833968E5bB786Ae419c4d13189fB081Cc43bab -keys_dir=$HOME/.thetacli/keys -msg=02f8a4c78085e8d4a51000f86ff86d942e833968e5 -encrypted=true
//
func main() {
	signerAddress, keysDir, message, encrypted := parseArguments()

	var keystore ks.Keystore
	var err error
	password := ""
	if encrypted {
		prompt := fmt.Sprintf("Please enter password: ")
		password, err = utils.GetPassword(prompt)
		if err != nil {
			panic(fmt.Sprintf("\n[ERROR] Failed to get password: %v\n", err))
		}
		keystore, err = ks.NewKeystoreEncrypted(keysDir, ks.StandardScryptN, ks.StandardScryptP)
	} else {
		keystore, err = ks.NewKeystorePlain(keysDir)
	}
	if err != nil {
		fmt.Printf("\n[ERROR] Failed to create keystore: %v\n", err)
		return
	}

	key, err := keystore.GetKey(signerAddress, password)
	if err != nil {
		fmt.Printf("\n[ERROR] Failed to get key: %v\n", err)
		return
	}

	msgHex, err := hex.DecodeString(message)
	if err != nil {
		fmt.Printf("\n[ERROR] message %v is not a hex string: %v\n", message, err)
		return
	}

	signature, err := key.Sign(msgHex)
	if err != nil {
		fmt.Printf("\n[ERROR] Failed sign the message: %v\n", err)
		return
	}

	fmt.Println("")
	fmt.Printf("--------------------------------------------------------------------------\n")
	fmt.Printf("Signature: %v\n", hex.EncodeToString(signature.ToBytes()))
	fmt.Printf("--------------------------------------------------------------------------\n")
	fmt.Println("")
}

func parseArguments() (signerAddress common.Address, keysDir, message string, encrypted bool) {
	signerAddressPtr := flag.String("signer", "", "the address of the signer")
	keysDirPtr := flag.String("keys_dir", "./keys", "the folder that contains the keys of the signers")
	messagePtr := flag.String("msg", "", "the message to be signed")
	encryptedPtr := flag.Bool("encrypted", true, "whether the private key is encrypted")

	flag.Parse()

	signerAddress = common.HexToAddress(*signerAddressPtr)
	keysDir = *keysDirPtr
	message = *messagePtr
	encrypted = *encryptedPtr
	return
}
