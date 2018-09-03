package types

import (
	"crypto/ecdsa"
	"fmt"
	"net"
	"strconv"

	"github.com/thetatoken/ukulele/crypto"
)

// --------------- Test Utilities --------------- //

// GetTestNetconn returns a net.Conn instance
func GetTestNetconn(port int) net.Conn {
	netconn, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		panic(fmt.Sprintf("Failed to create a net connection: %v", err))
	}
	return netconn
}

// GetTestListener returns a net.Listener instance
func GetTestListener(port int) net.Listener {
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		panic(fmt.Sprintf("Failed to listen: %v", err))
	}

	return listener
}

// GetTestRandPubKey returns a randomly generated public key
func GetTestRandPubKey() ecdsa.PublicKey {
	randPrivKey, err := crypto.GenerateKey()
	if err != nil {
		panic(fmt.Sprintf("Failed to generate a random private key: %v", err))
	}
	return randPrivKey.PublicKey
}
