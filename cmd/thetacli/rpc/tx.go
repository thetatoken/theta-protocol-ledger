package rpc

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"

	"github.com/spf13/viper"
	rpcc "github.com/ybbus/jsonrpc"

	"github.com/thetatoken/theta/cmd/thetacli/cmd/utils"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/ledger/types"
	trpc "github.com/thetatoken/theta/rpc"
)

// ------------------------------- SendTx -----------------------------------

type SendArgs struct {
	ChainID  string `json:"chain_id"`
	From     string `json:"from"`
	To       string `json:"to"`
	ThetaWei string `json:"thetawei"`
	TFuelWei string `json:"tfuelwei"`
	Fee      string `json:"fee"`
	Sequence string `json:"sequence"`
	Async    bool   `json:"async"`
}

type SendResult struct {
	TxHash string            `json:"hash"`
	Block  *core.BlockHeader `json:"block",rlp:"nil"`
}

func (t *ThetaCliRPCService) Send(args *SendArgs, result *SendResult) (err error) {
	if len(args.From) == 0 || len(args.To) == 0 {
		return fmt.Errorf("The from and to address cannot be empty")
	}
	if args.From == args.To {
		return fmt.Errorf("The from and to address cannot be identical")
	}

	from := common.HexToAddress(args.From)
	to := common.HexToAddress(args.To)
	thetawei, ok := new(big.Int).SetString(args.ThetaWei, 10)
	if !ok {
		return fmt.Errorf("Failed to parse thetawei: %v", args.ThetaWei)
	}
	tfuelwei, ok := new(big.Int).SetString(args.TFuelWei, 10)
	if !ok {
		return fmt.Errorf("Failed to parse tfuelwei: %v", args.TFuelWei)
	}
	fee, ok := new(big.Int).SetString(args.Fee, 10)
	if !ok {
		return fmt.Errorf("Failed to parse fee: %v", args.Fee)
	}
	sequence, err := strconv.ParseUint(args.Sequence, 10, 64)
	if err != nil {
		return err
	}

	if !t.wallet.IsUnlocked(from) {
		return fmt.Errorf("The from address %v has not been unlocked yet", from.Hex())
	}

	inputs := []types.TxInput{{
		Address: from,
		Coins: types.Coins{
			TFuelWei: new(big.Int).Add(tfuelwei, fee),
			ThetaWei: thetawei,
		},
		Sequence: sequence,
	}}
	outputs := []types.TxOutput{{
		Address: to,
		Coins: types.Coins{
			TFuelWei: tfuelwei,
			ThetaWei: thetawei,
		},
	}}
	sendTx := &types.SendTx{
		Fee: types.Coins{
			ThetaWei: new(big.Int).SetUint64(0),
			TFuelWei: fee,
		},
		Inputs:  inputs,
		Outputs: outputs,
	}

	signBytes := sendTx.SignBytes(args.ChainID)
	sig, err := t.wallet.Sign(from, signBytes)
	if err != nil {
		utils.Error("Failed to sign transaction: %v\n", err)
	}
	sendTx.SetSignature(from, sig)

	raw, err := types.TxToBytes(sendTx)
	if err != nil {
		utils.Error("Failed to encode transaction: %v\n", err)
	}
	signedTx := hex.EncodeToString(raw)

	client := rpcc.NewRPCClient(viper.GetString(utils.CfgRemoteRPCEndpoint))

	rpcMethod := "theta.BroadcastRawTransaction"
	if args.Async {
		rpcMethod = "theta.BroadcastRawTransactionAsync"
	}
	res, err := client.Call(rpcMethod, trpc.BroadcastRawTransactionArgs{TxBytes: signedTx})
	if err != nil {
		return err
	}
	if res.Error != nil {
		return fmt.Errorf("Server returned error: %v", res.Error)
	}
	trpcResult := &trpc.BroadcastRawTransactionResult{}
	err = res.GetObject(trpcResult)
	if err != nil {
		return fmt.Errorf("Failed to parse Theta node response: %v", err)
	}

	result.TxHash = trpcResult.TxHash
	result.Block = trpcResult.Block

	return nil
}
