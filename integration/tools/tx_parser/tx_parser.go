package main

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/thetatoken/theta/ledger/types"
)

func handleError(err error) {
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: tx_parser <tx_HEX>")
}

func main() {
	args := os.Args[1:]
	if len(args) != 1 {
		printUsage()
		return
	}

	raw, err := hex.DecodeString(args[0])
	handleError(err)

	tx, err := types.TxFromBytes(raw)
	handleError(err)

	fmt.Printf("\n%#v\n\n", tx)

}
