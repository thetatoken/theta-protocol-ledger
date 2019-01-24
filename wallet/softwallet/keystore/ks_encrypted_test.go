// Adapted for Theta
// Copyright 2016 The go-ethereum Authors
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
	"crypto/rand"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/thetatoken/theta/common"
)

func Test_PBKDF2_1(t *testing.T) {
	t.Parallel()
	tests := loadKeyStoreTest("testdata/test_vector.json", t)
	testDecrypt(tests["wikipage_test_vector_pbkdf2"], t)
}

func Test_Scrypt_1(t *testing.T) {
	t.Parallel()
	tests := loadKeyStoreTest("testdata/test_vector.json", t)
	testDecrypt(tests["wikipage_test_vector_scrypt"], t)
}

func Test_31_Byte_Key(t *testing.T) {
	t.Parallel()
	tests := loadKeyStoreTest("testdata/test_vector.json", t)
	testDecrypt(tests["31_byte_key"], t)
}

func Test_30_Byte_Key(t *testing.T) {
	t.Parallel()
	tests := loadKeyStoreTest("testdata/test_vector.json", t)
	testDecrypt(tests["30_byte_key"], t)
}

func TestKeyStoreEncrypted(t *testing.T) {
	dir, ks := tmpKeyStoreIface(t, true)
	defer os.RemoveAll(dir)

	pass := "foo"
	k1, err := storeNewKeyTest(ks, rand.Reader, pass)
	if err != nil {
		t.Fatal(err)
	}
	k2, err := ks.GetKey(k1.Address, pass)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(k1.Address, k2.Address) {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(k1.PrivateKey, k2.PrivateKey) {
		t.Fatal(err)
	}
}

func TestKeyStoreEncryptedDecryptionFail(t *testing.T) {
	dir, ks := tmpKeyStoreIface(t, true)
	defer os.RemoveAll(dir)

	pass := "foo"
	k1, err := storeNewKeyTest(ks, rand.Reader, pass)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ks.GetKey(k1.Address, "bar"); err != ErrDecrypt {
		t.Fatalf("wrong error for invalid password\ngot %q\nwant %q", err, ErrDecrypt)
	}
}

// Tests that a json key file can be decrypted and encrypted in multiple rounds.
func TestKeyEncryptDecrypt(t *testing.T) {
	keyjson, err := ioutil.ReadFile("testdata/very-light-scrypt.json")
	if err != nil {
		t.Fatal(err)
	}
	password := ""
	address := common.HexToAddress("45dea0fb0bba44f4fcf290bba71fd57d7117cbb8")

	// Do a few rounds of decryption and encryption
	for i := 0; i < 3; i++ {
		// Try a bad password first
		if _, err := decryptKey(keyjson, password+"bad"); err == nil {
			t.Errorf("test %d: json key decrypted with bad password", i)
		}
		// Decrypt with the correct password
		key, err := decryptKey(keyjson, password)
		if err != nil {
			t.Fatalf("test %d: json key failed to decrypt: %v", i, err)
		}
		if key.Address != address {
			t.Errorf("test %d: key address mismatch: have %x, want %x", i, key.Address, address)
		}
		// Recrypt with a new password and start over
		password += "new data appended"
		if keyjson, err = encryptKey(key, password, veryLightScryptN, veryLightScryptP); err != nil {
			t.Errorf("test %d: failed to recrypt key %v", i, err)
		}
	}
}
