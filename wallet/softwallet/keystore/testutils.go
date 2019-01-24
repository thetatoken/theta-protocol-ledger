// Adapted for Theta
// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package keystore

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/thetatoken/theta/crypto"
)

const (
	veryLightScryptN = 2
	veryLightScryptP = 1
)

func tmpKeyStoreIface(t *testing.T, encrypted bool) (dir string, ks Keystore) {
	d, err := ioutil.TempDir("", "theta-keystore-test")
	if err != nil {
		t.Fatal(err)
	}
	if encrypted {
		ks, err = NewKeystoreEncrypted(d, veryLightScryptN, veryLightScryptP)
	} else {
		ks, err = NewKeystorePlain(d)
	}
	if err != nil {
		t.Fatal(err)
	}
	return d, ks
}

func storeNewKeyTest(ks Keystore, rand io.Reader, auth string) (*Key, error) {
	privKey, _, err := crypto.GenerateKeyPair()
	key := NewKey(privKey)
	if err != nil {
		return nil, err
	}
	if err := ks.StoreKey(key, auth); err != nil {
		return nil, err
	}
	return key, err
}

func loadKeyStoreTest(file string, t *testing.T) map[string]KeyStoreTest {
	tests := make(map[string]KeyStoreTest)
	err := loadJSONTest(file, &tests)
	if err != nil {
		t.Fatal(err)
	}
	return tests
}

func loadJSONTest(file string, val interface{}) error {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(content, val); err != nil {
		if syntaxerr, ok := err.(*json.SyntaxError); ok {
			line := findLine(content, syntaxerr.Offset)
			return fmt.Errorf("JSON syntax error at %v:%v: %v", file, line, err)
		}
		return fmt.Errorf("JSON unmarshal error in %v: %v", file, err)
	}
	return nil
}

func findLine(data []byte, offset int64) (line int) {
	line = 1
	for i, r := range string(data) {
		if int64(i) >= offset {
			return
		}
		if r == '\n' {
			line++
		}
	}
	return
}

func testDecrypt(test KeyStoreTest, t *testing.T) {
	keyjson, err := json.Marshal(test.Json)
	if err != nil {
		t.Fatal(err)
	}
	key, err := decryptKey(keyjson, test.Password)
	if err != nil {
		t.Fatal(err)
	}
	privHex := hex.EncodeToString(bytes.Trim(key.PrivateKey.ToBytes(), "\x00"))
	if test.Priv != privHex {
		t.Fatal(fmt.Errorf("Decrypted bytes not equal to test, expected %v have %v", test.Priv, privHex))
	}
}

var testsSubmodule = filepath.Join("..", "..", "tests", "testdata", "KeyStoreTests")

func skipIfSubmoduleMissing(t *testing.T) {
	if !fileExist(testsSubmodule) {
		t.Skipf("can't find JSON tests from submodule at %s", testsSubmodule)
	}
}

func fileExist(filePath string) bool {
	_, err := os.Stat(filePath)
	if err != nil && os.IsNotExist(err) {
		return false
	}

	return true
}

// Test and utils for the key store tests in the Ethereum JSON tests;
// testdataKeyStoreTests/basic_tests.json
type KeyStoreTest struct {
	Json     encryptedKeyJSON
	Password string
	Priv     string
}
