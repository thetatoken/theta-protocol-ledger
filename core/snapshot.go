package core

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/thetatoken/ukulele/common"
)

const (
	SVStart = iota
	SVEnd
)

type SnapshotTrieRecord struct {
	K common.Bytes // key
	V common.Bytes // value
}

type SnapshotFirstBlock struct {
	Header BlockHeader
	Proof  VCPProof
}

type SnapshotThirdBlock struct {
	Header BlockHeader
	Votes  []Vote
}

type SnapshotBlockTrio struct {
	First  SnapshotFirstBlock
	Second BlockHeader
	Third  SnapshotThirdBlock
}

type SnapshotMetadata struct {
	BlockTrios []SnapshotBlockTrio
}

////////////////////////////////////////

type proofKV struct {
	key []byte
	val []byte
}

type VCPProof struct {
	kvs []*proofKV
}

func (vp *VCPProof) Get(key []byte) (value []byte, err error) {
	for _, kv := range vp.kvs {
		if bytes.Compare(key, kv.key) == 0 {
			return kv.val, nil
		}
	}
	return nil, fmt.Errorf("key %v does not exist", hex.EncodeToString(key))
}

func (vp *VCPProof) Has(key []byte) (bool, error) {
	for _, kv := range vp.kvs {
		if bytes.Compare(key, kv.key) == 0 {
			return true, nil
		}
	}
	return false, fmt.Errorf("key %v does not exist", hex.EncodeToString(key))
}

func (vp *VCPProof) Put(key []byte, value []byte) error {
	for _, kv := range vp.kvs {
		if bytes.Compare(key, kv.key) == 0 {
			kv.val = value
			return nil
		}
	}
	vp.kvs = append(vp.kvs, &proofKV{key, value})
	return nil
}
