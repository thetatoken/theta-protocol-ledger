package types

import "crypto/ecdsa"

//
// NodeInfo provides the information of the current node
//
type NodeInfo struct {
	PriKey ecdsa.PrivateKey
	PubKey ecdsa.PublicKey
}
