package keystore

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"reflect"
	"testing"
)

func TestKeyStorePlain(t *testing.T) {
	dir, ks := tmpKeyStoreIface(t, false)
	defer os.RemoveAll(dir)

	pass := "" // not used but required by API
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

	t.Logf("k1.Id = %v", k1.Id)
	t.Logf("k2.Id = %v", k2.Id)

	t.Logf("k1.Address = %v", k1.Address.Hex())
	t.Logf("k2.Address = %v", k2.Address.Hex())

	t.Logf("k1.PrivateKey = %v", hex.EncodeToString(k1.PrivateKey.ToBytes()))
	t.Logf("k2.PrivateKey = %v", hex.EncodeToString(k2.PrivateKey.ToBytes()))
}
