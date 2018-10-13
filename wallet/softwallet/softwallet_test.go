package softwallet

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
)

func TestPlainSoftWalletBasics(t *testing.T) {
	testSoftWalletBasics(t, KeystoreTypePlain)
}

func TestEncryptedSoftWalletBasics(t *testing.T) {
	testSoftWalletBasics(t, KeystoreTypeEncrypted)
}

// ---------------- Test Utilities ---------------- //

func testSoftWalletBasics(t *testing.T, ksType KeystoreType) {
	assert := assert.New(t)

	tmpdir := createTempDir()
	defer os.RemoveAll(tmpdir)

	wallet, err := NewSoftWallet(tmpdir, ksType)
	assert.Nil(err)
	assert.Equal(0, len(wallet.List()))

	password1 := "abcd"
	addr, err := wallet.NewKey(password1)
	assert.NotEqual(common.Address{}, addr)
	assert.Nil(err)
	assert.Equal(1, len(wallet.List())) // the newly generated key should be unlocked

	err = wallet.Close(addr)
	assert.Nil(err)
	assert.Equal(0, len(wallet.List()))

	password2 := "xyz123"
	err = wallet.UpdatePassword(addr, password1, password2) // update password for a closed password
	assert.Nil(err)

	err = wallet.Unlock(addr, password1)
	if ksType == KeystoreTypeEncrypted {
		assert.NotNil(err)
	}
	err = wallet.Unlock(addr, password2)
	assert.Nil(err)
	err = wallet.Close(addr)
	assert.Nil(err)

	password3 := "Kdaw82892fDWO"
	err = wallet.Unlock(addr, password2)
	assert.Nil(err)
	err = wallet.UpdatePassword(addr, password2, password3) // update password for a unlocked password
	assert.Nil(err)

	signature, err := wallet.Sign(addr, common.Bytes("hello world"))
	assert.False(signature.IsEmpty())
	assert.Nil(err)

	err = wallet.Close(addr)
	assert.Nil(err)

	signature, err = wallet.Sign(addr, common.Bytes("hello world"))
	assert.Nil(signature)
	assert.NotNil(err)

	err = wallet.Delete(addr, password2)
	if ksType == KeystoreTypeEncrypted {
		assert.NotNil(err)
		err = wallet.Delete(addr, password3)
		assert.Nil(err)
	} else {
		assert.Nil(err)
	}
	err = wallet.Unlock(addr, password3)
	assert.NotNil(err)

}

func createTempDir() string {
	dir, err := ioutil.TempDir("", "theta-softwallet-test")
	if err != nil {
		panic(err)
	}

	return dir
}
