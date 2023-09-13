package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/ledger/types"
	"github.com/thetatoken/theta/rlp"
)

func handleError(err error) {
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: hex_obj_parser <obj_hex>")
}

func txFromBytes(raw []byte) error {
	tx, err := types.TxFromBytes(raw)
	if err == nil {
		fmt.Printf("\nTx: %v\n\n", tx)
		if jsonStr, err := json.MarshalIndent(tx, "", "    "); err == nil {
			fmt.Printf("\nJSON: %s\n", jsonStr)
		} else {
			fmt.Printf("\nJSON: %v\n", err)
		}
		return nil
	}
	return fmt.Errorf("Not a transaction")
}

func voteFromBytes(raw []byte) error {
	vote := core.Vote{}
	err := rlp.DecodeBytes(raw, &vote)
	if err == nil {
		fmt.Printf("\nVote: %v\n\n", vote)
		return nil
	}
	return fmt.Errorf("Not a vote object")
}

func blockFromBytes(raw []byte) error {
	block := core.ExtendedBlock{}
	err := rlp.DecodeBytes(raw, &block)
	if err == nil {
		fmt.Printf("\nBlock: %v\n\n", block)
		return nil
	}
	return fmt.Errorf("Not an extended block object")
}

func ethTxFromBytes(raw []byte) error {
	ethTx := types.EthTransaction{}
	err := rlp.DecodeBytes(raw, &ethTx)
	if err == nil {
		fmt.Printf("\nEthTransaction: %v\n\n", ethTx)
		return nil
	}
	return fmt.Errorf("Not an ETH transaction object")
}

func main() {
	args := os.Args[1:]
	if len(args) != 1 {
		printUsage()
		return
	}

	objHexStr := args[0]
	if strings.HasPrefix(objHexStr, "0x") {
		objHexStr = objHexStr[2:]
	}

	raw, err := hex.DecodeString(objHexStr)
	handleError(err)

	handlers := []func(raw []byte) error{
		blockFromBytes,
		voteFromBytes,
		txFromBytes,
		ethTxFromBytes,
	}
	for _, handler := range handlers {
		err := handler(raw)
		if err == nil {
			return
		}
	}
	err = fmt.Errorf("Unable to identity the type of the object")
	handleError(err)
}
