package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"path/filepath"

	"github.com/thetatoken/theta/crypto"
	ks "github.com/thetatoken/theta/wallet/softwallet/keystore"
)

//
// Usage: encrypt_sk -password=<password> -sk=<private_key_hex>
//
func main() {
	keysDirPathPtr := flag.String("dir", "./", "the folder to generate the encrypted key file")
	passwordPtr := flag.String("password", "", "the password for the private key")
	skHexStrPtr := flag.String("sk", "", "the private key to be encrypted")

	flag.Parse()

	keysDirPath := *keysDirPathPtr
	password := *passwordPtr
	skHexStr := *skHexStrPtr

	skBytes, err := hex.DecodeString(skHexStr)
	if err != nil {
		fmt.Printf("Failed to decode private key hex string: %v\n", err)
		return
	}

	sk, err := crypto.PrivateKeyFromBytes(skBytes)
	if err != nil {
		fmt.Printf("Failed to parse private key bytes: %v\n", err)
		return
	}

	encryptedKeystore, err := ks.NewKeystoreEncrypted(keysDirPath, ks.StandardScryptN, ks.StandardScryptP)
	if err != nil {
		fmt.Printf("Failed to create key store: %v\n", err)
		return
	}

	key := ks.NewKey(sk)
	err = encryptedKeystore.StoreKey(key, password)
	if err != nil {
		fmt.Printf("Failed to encrypt the private key: %v\n", err)
		return
	}

	address := sk.PublicKey().Address()
	encryptedKeyFilePath := filepath.Join(keysDirPath, "encrypted", address.Hex()[2:])
	fmt.Printf("Private key successfully encrypted: %v\n", encryptedKeyFilePath)
}
