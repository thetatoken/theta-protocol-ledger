package types

// Helper functions for testing

import (
	"fmt"
	"math/rand"

	"github.com/thetatoken/ukulele/crypto"
)

type PrivAccount struct {
	PrivKey crypto.PrivateKey
	Account Account
}

func (pa *PrivAccount) Sign(msg []byte) crypto.Signature {
	sig, err := pa.PrivKey.Sign(msg)
	if err != nil {
		panic(fmt.Sprintf("Failed to sign message \"%v\": %v", msg, err))
	}
	return sig
}

// Creates a PrivAccount from secret.
// The amount is not set.
func PrivAccountFromSecret(secret string) PrivAccount {
	privKey, _, err := crypto.GenerateKeyPair(crypto.CrytoSchemeECDSA)
	if err != nil {
		panic(fmt.Sprintf("Failed to generate private key: %v", err))
	}
	privAccount := PrivAccount{
		PrivKey: privKey,
		Account: Account{
			PubKey:                 privKey.PublicKey(),
			LastUpdatedBlockHeight: 1,
		},
	}
	return privAccount
}

// Make `num` random accounts
func RandAccounts(num int, minAmount int64, maxAmount int64) []PrivAccount {
	privAccs := make([]PrivAccount, num)
	for i := 0; i < num; i++ {

		balance := minAmount
		if maxAmount > minAmount {
			balance += rand.Int63() % (maxAmount - minAmount)
		}

		privKey, _, err := crypto.GenerateKeyPair(crypto.CrytoSchemeECDSA)
		if err != nil {
			panic(fmt.Sprintf("Failed to generate key pair: %v", err))
		}
		pubKey := privKey.PublicKey()
		privAccs[i] = PrivAccount{
			PrivKey: privKey,
			Account: Account{
				PubKey:                 pubKey,
				Balance:                Coins{Coin{"GammaWei", balance}, Coin{"ThetaWei", balance}},
				LastUpdatedBlockHeight: 1,
			},
		}
	}

	return privAccs
}

/////////////////////////////////////////////////////////////////

//func MakeAccs(secrets ...string) (accs []PrivAccount) {
//	for _, secret := range secrets {
//		privAcc := PrivAccountFromSecret(secret)
//		privAcc.Account.Balance = Coins{{"mycoin", 7}}
//		accs = append(accs, privAcc)
//	}
//	return
//}

func MakeAcc(secret string) PrivAccount {
	privAcc := PrivAccountFromSecret(secret)
	privAcc.Account.Balance = Coins{Coin{"GammaWei", 5}, {"ThetaWei", 7}}
	return privAcc
}

func Accs2TxInputs(seq int, accs ...PrivAccount) []TxInput {
	var txs []TxInput
	for _, acc := range accs {
		tx := NewTxInput(
			acc.Account.PubKey,
			Coins{Coin{"GammaWei", 1}, {"ThetaWei", 4}},
			seq)
		txs = append(txs, tx)
	}
	return txs
}

//turn a list of accounts into basic list of transaction outputs
func Accs2TxOutputs(accs ...PrivAccount) []TxOutput {
	var txs []TxOutput
	for _, acc := range accs {
		tx := TxOutput{
			Address: acc.Account.PubKey.Address(),
			Coins:   Coins{{"ThetaWei", 4}},
		}
		txs = append(txs, tx)
	}
	return txs
}

func MakeSendTx(seq int, accOut PrivAccount, accsIn ...PrivAccount) *SendTx {
	tx := &SendTx{
		Gas:     0,
		Fee:     Coin{"GammaWei", 1},
		Inputs:  Accs2TxInputs(seq, accsIn...),
		Outputs: Accs2TxOutputs(accOut),
	}

	return tx
}

func SignTx(chainID string, tx *SendTx, accs ...PrivAccount) {
	signBytes := tx.SignBytes(chainID)
	for i, _ := range tx.Inputs {
		tx.Inputs[i].Signature = accs[i].Sign(signBytes)
	}
}
