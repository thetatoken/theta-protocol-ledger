package types

// Helper functions for testing

import (
	"fmt"
	"math/big"
	"math/rand"

	"github.com/thetatoken/theta/crypto"
)

type PrivAccount struct {
	PrivKey *crypto.PrivateKey
	Account
}

func (pa *PrivAccount) Sign(msg []byte) *crypto.Signature {
	sig, err := pa.PrivKey.Sign(msg)
	if err != nil {
		panic(fmt.Sprintf("Failed to sign message \"%v\": %v", msg, err))
	}
	return sig
}

// Creates a PrivAccount from secret.
// The amount is not set.
func PrivAccountFromSecret(secret string) PrivAccount {
	privKey, _, err := crypto.TEST_GenerateKeyPairWithSeed(secret)
	if err != nil {
		panic(fmt.Sprintf("Failed to generate private key: %v", err))
	}
	privAccount := PrivAccount{
		PrivKey: privKey,
		Account: Account{
			Address:                privKey.PublicKey().Address(),
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

		privKey, _, err := crypto.GenerateKeyPair()
		if err != nil {
			panic(fmt.Sprintf("Failed to generate key pair: %v", err))
		}
		pubKey := privKey.PublicKey()
		privAccs[i] = PrivAccount{
			PrivKey: privKey,
			Account: Account{
				Address:                pubKey.Address(),
				Balance:                Coins{TFuelWei: big.NewInt(balance), ThetaWei: big.NewInt(balance)},
				LastUpdatedBlockHeight: 1,
			},
		}
	}

	return privAccs
}

/////////////////////////////////////////////////////////////////

func MakeAcc(secret string) PrivAccount {
	privAcc := MakeAccWithInitBalance(secret, NewCoins(7*10e12, 5*10e12))
	return privAcc
}

func MakeAccWithInitBalance(secret string, initBalance Coins) PrivAccount {
	privAcc := PrivAccountFromSecret(secret)
	privAcc.Account.Balance = initBalance
	return privAcc
}

func Accs2TxInputs(seq int, accs ...PrivAccount) []TxInput {
	var txs []TxInput
	for _, acc := range accs {
		tx := NewTxInput(
			acc.Account.Address,
			NewCoins(4, int64(MinimumTransactionFeeTFuelWeiJune2021)),
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
			Address: acc.Account.Address,
			Coins:   NewCoins(4, 0),
		}
		txs = append(txs, tx)
	}
	return txs
}

func MakeSendTx(seq int, accOut PrivAccount, accsIn ...PrivAccount) *SendTx {
	tx := &SendTx{
		Fee:     NewCoins(0, int64(MinimumTransactionFeeTFuelWeiJune2021)),
		Inputs:  Accs2TxInputs(seq, accsIn...),
		Outputs: Accs2TxOutputs(accOut),
	}

	return tx
}

func SignSendTx(chainID string, tx *SendTx, accs ...PrivAccount) {
	signBytes := tx.SignBytes(chainID)
	for i, _ := range tx.Inputs {
		tx.Inputs[i].Signature = accs[i].Sign(signBytes)
	}
}
