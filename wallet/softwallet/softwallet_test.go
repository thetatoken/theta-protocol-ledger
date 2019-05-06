package softwallet

import (
	"io/ioutil"
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/common"
)

func TestPlainSoftWalletBasics(t *testing.T) {
	testSoftWalletBasics(t, KeystoreTypePlain)
}

func TestEncryptedSoftWalletBasics(t *testing.T) {
	testSoftWalletBasics(t, KeystoreTypeEncrypted)
}

func TestPlainSoftWalletMultipleKeys(t *testing.T) {
	testSoftWalletMultipleKeys(t, KeystoreTypePlain)
}

func TestEncryptedSoftWalletMultipleKeys(t *testing.T) {
	testSoftWalletMultipleKeys(t, KeystoreTypeEncrypted)
}

// ---------------- Test Utilities ---------------- //

func testSoftWalletBasics(t *testing.T, ksType KeystoreType) {
	assert := assert.New(t)

	tmpdir := createTempDir()
	defer os.RemoveAll(tmpdir)

	wallet, err := NewSoftWallet(tmpdir, ksType)
	assert.Nil(err)
	addrs, err := wallet.List()
	assert.Nil(err)
	assert.Equal(0, len(addrs))

	password1 := "abcd"
	addr, err := wallet.NewKey(password1)
	assert.NotEqual(common.Address{}, addr)
	assert.Nil(err)
	addrs, err = wallet.List()
	assert.Nil(err)
	assert.Equal(1, len(addrs))
	assert.Equal([]common.Address{addr}, addrs)

	err = wallet.Lock(addr)
	assert.Nil(err)
	addrs, err = wallet.List()
	assert.Nil(err)
	assert.Equal(1, len(addrs))
	assert.Equal([]common.Address{addr}, addrs)

	password2 := "xyz123"
	err = wallet.UpdatePassword(addr, password1, password2) // update password for a closed password
	assert.Nil(err)

	err = wallet.Unlock(addr, password1, nil)
	if ksType == KeystoreTypeEncrypted {
		assert.NotNil(err)
	}
	err = wallet.Unlock(addr, password2, nil)
	assert.Nil(err)
	err = wallet.Lock(addr)
	assert.Nil(err)

	password3 := "Kdaw82892fDWO"
	err = wallet.Unlock(addr, password2, nil)
	assert.Nil(err)
	err = wallet.UpdatePassword(addr, password2, password3) // update password for a unlocked password
	assert.Nil(err)

	signature, err := wallet.Sign(addr, common.Bytes("hello world"))
	assert.False(signature.IsEmpty())
	assert.Nil(err)

	err = wallet.Lock(addr)
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
	err = wallet.Unlock(addr, password3, nil)
	assert.NotNil(err)
}

func testSoftWalletMultipleKeys(t *testing.T, ksType KeystoreType) {
	assert := assert.New(t)

	tmpdir := createTempDir()
	defer os.RemoveAll(tmpdir)

	wallet, err := NewSoftWallet(tmpdir, ksType)
	assert.Nil(err)
	addrs, err := wallet.List()
	assert.Nil(err)
	assert.Equal(0, len(addrs))

	password1 := "password1"
	addr1, err := wallet.NewKey(password1)
	assert.NotEqual(common.Address{}, addr1)
	assert.Nil(err)
	addrs, err = wallet.List()
	assert.Nil(err)
	assert.Equal(1, len(addrs))
	assert.Equal([]common.Address{addr1}, addrs)

	password2 := "password2"
	addr2, err := wallet.NewKey(password2)
	assert.NotEqual(common.Address{}, addr2)
	assert.Nil(err)
	addrs, err = wallet.List()
	assert.Nil(err)
	assert.Equal(2, len(addrs))
	assert.Equal(sortAddresses([]common.Address{addr1, addr2}), sortAddresses(addrs))

	password3 := "password3"
	addr3, err := wallet.NewKey(password3)
	assert.NotEqual(common.Address{}, addr3)
	assert.Nil(err)
	addrs, err = wallet.List()
	assert.Nil(err)
	assert.Equal(3, len(addrs))
	assert.Equal(sortAddresses([]common.Address{addr1, addr2, addr3}), sortAddresses(addrs))

	password4 := "password4"
	addr4, err := wallet.NewKey(password4)
	assert.NotEqual(common.Address{}, addr4)
	assert.Nil(err)
	addrs, err = wallet.List()
	assert.Nil(err)
	assert.Equal(4, len(addrs))
	assert.Equal(sortAddresses([]common.Address{addr1, addr2, addr3, addr4}), sortAddresses(addrs))

	err = wallet.Lock(addr1)
	assert.Nil(err)
	addrs, err = wallet.List()
	assert.Nil(err)
	assert.Equal(4, len(addrs))
	assert.Equal(sortAddresses([]common.Address{addr1, addr2, addr3, addr4}), sortAddresses(addrs))

	err = wallet.Delete(addr3, password3)
	assert.Nil(err)
	addrs, err = wallet.List()
	assert.Nil(err)
	assert.Equal(3, len(addrs))
	assert.Equal(sortAddresses([]common.Address{addr1, addr2, addr4}), sortAddresses(addrs))

	signature3, err := wallet.Sign(addr3, common.Bytes("hello world"))
	assert.Nil(signature3)
	assert.NotNil(err)

	signature2, err := wallet.Sign(addr2, common.Bytes("hello world"))
	assert.NotNil(signature2)
	assert.Nil(err)

	signature4, err := wallet.Sign(addr4, common.Bytes("hello world"))
	assert.NotNil(signature4)
	assert.Nil(err)

	assert.NotEqual(signature2.ToBytes(), signature4.ToBytes())

	err = wallet.Delete(addr2, password2)
	assert.Nil(err)
	addrs, err = wallet.List()
	assert.Nil(err)
	assert.Equal(2, len(addrs))
	assert.Equal(sortAddresses([]common.Address{addr1, addr4}), sortAddresses(addrs))
}

func createTempDir() string {
	dir, err := ioutil.TempDir("", "theta-softwallet-test")
	if err != nil {
		panic(err)
	}

	return dir
}

func sortAddresses(addresses []common.Address) []common.Address {
	sortedAddresses := addresses
	sort.Slice(sortedAddresses[:], func(i, j int) bool {
		return sortedAddresses[i].Hex() < sortedAddresses[j].Hex()
	})
	return sortedAddresses
}
