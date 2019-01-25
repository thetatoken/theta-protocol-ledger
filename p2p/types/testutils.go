package types

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/thetatoken/theta/crypto"
)

// --------------- Test Utilities --------------- //

// GetTestNetconn returns a net.Conn instance
func GetTestNetconn(port int) net.Conn {
	numRetries := 5
	var err error
	var netconn net.Conn
	for i := 0; i < numRetries; i++ {
		netconn, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(port))
		if err == nil {
			return netconn
		}
		time.Sleep(50 * time.Millisecond)
	}
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
func GetTestRandPubKey() *crypto.PublicKey {
	_, randPubKey, err := crypto.GenerateKeyPair()
	if err != nil {
		panic(fmt.Sprintf("Failed to generate a random private key: %v", err))
	}
	return randPubKey
}
