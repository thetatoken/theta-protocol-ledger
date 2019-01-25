package core

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/log"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/rlp"
)

const BlockTrioStoreKeyPrefix = "blocktrio_"
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

type SnapshotSecondBlock struct {
	Header BlockHeader
}

type SnapshotThirdBlock struct {
	Header  BlockHeader
	VoteSet *VoteSet
}

type SnapshotBlockTrio struct {
	First  SnapshotFirstBlock
	Second SnapshotSecondBlock
	Third  SnapshotThirdBlock
}

type SnapshotMetadata struct {
	BlockTrios []SnapshotBlockTrio
}

func WriteMetadata(writer *bufio.Writer, metadata *SnapshotMetadata) error {
	raw, err := rlp.EncodeToBytes(*metadata)
	if err != nil {
		log.Error("Failed to encode snapshot metadata")
		return err
	}
	// write length first
	_, err = writer.Write(Itobytes(uint64(len(raw))))
	if err != nil {
		log.Error("Failed to write snapshot metadata length")
		return err
	}
	// write metadata itself
	_, err = writer.Write(raw)
	if err != nil {
		log.Error("Failed to write snapshot metadata")
		return err
	}

	meta := &SnapshotMetadata{}
	rlp.DecodeBytes(raw, meta)

	return nil
}

func WriteRecord(writer *bufio.Writer, k, v common.Bytes) error {
	record := SnapshotTrieRecord{K: k, V: v}
	raw, err := rlp.EncodeToBytes(record)
	if err != nil {
		return fmt.Errorf("Failed to encode storage record, %v", err)
	}
	// write length first
	_, err = writer.Write(Itobytes(uint64(len(raw))))
	if err != nil {
		return fmt.Errorf("Failed to write storage record length, %v", err)
	}
	// write record itself
	_, err = writer.Write(raw)
	if err != nil {
		return fmt.Errorf("Failed to write storage record, %v", err)
	}
	err = writer.Flush()
	if err != nil {
		return fmt.Errorf("Failed to flush storage record, %v", err)
	}
	return nil
}

func ReadRecord(file *os.File, obj interface{}) error {
	sizeBytes := make([]byte, 8)
	n, err := io.ReadAtLeast(file, sizeBytes, 8)
	if err != nil {
		return err
	}
	if n < 8 {
		return fmt.Errorf("Failed to read record length")
	}
	size := Bytestoi(sizeBytes)
	bytes := make([]byte, size)
	n, err = io.ReadAtLeast(file, bytes, int(size))
	if err != nil {
		return err
	}
	if uint64(n) < size {
		return fmt.Errorf("Failed to read record, %v < %v", n, size)
	}
	err = rlp.DecodeBytes(bytes, obj)
	return nil
}

func Bytestoi(arr []byte) uint64 {
	return binary.LittleEndian.Uint64(arr)
}

func Itobytes(val uint64) []byte {
	arr := make([]byte, 8)
	binary.LittleEndian.PutUint64(arr, val)
	return arr
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
