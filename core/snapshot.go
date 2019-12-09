package core

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/rlp"
)

const BlockTrioStoreKeyPrefix = "prooftrio_"
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
	ProofTrios []SnapshotBlockTrio
	TailTrio   SnapshotBlockTrio
}

func WriteMetadata(writer *bufio.Writer, metadata *SnapshotMetadata) error {
	raw, err := rlp.EncodeToBytes(metadata)
	if err != nil {
		logger.Error("Failed to encode snapshot metadata")
		return err
	}
	// write length first
	_, err = writer.Write(Itobytes(uint64(len(raw))))
	if err != nil {
		logger.Error("Failed to write snapshot metadata length")
		return err
	}
	// write metadata itself
	_, err = writer.Write(raw)
	if err != nil {
		logger.Error("Failed to write snapshot metadata")
		return err
	}
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
	Key []byte
	Val []byte
}

type VCPProof struct {
	kvs []*proofKV
}

func (vp VCPProof) GetKvs() []*proofKV {
	return vp.kvs
}

var _ rlp.Encoder = (*VCPProof)(nil)

// EncodeRLP implements RLP Encoder interface.
func (vp VCPProof) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, vp.GetKvs())
}

var _ rlp.Decoder = (*VCPProof)(nil)

// DecodeRLP implements RLP Decoder interface.
func (vp *VCPProof) DecodeRLP(stream *rlp.Stream) error {
	proof := []*proofKV{}
	err := stream.Decode(&proof)
	if err != nil {
		return err
	}
	vp.kvs = []*proofKV{}
	for _, kv := range proof {
		vp.kvs = append(vp.kvs, kv)
	}
	return nil
}

func (vp *VCPProof) Get(key []byte) (value []byte, err error) {
	for _, kv := range vp.kvs {
		if bytes.Compare(key, kv.Key) == 0 {
			return kv.Val, nil
		}
	}
	return nil, fmt.Errorf("key %v does not exist", hex.EncodeToString(key))
}

func (vp *VCPProof) Has(key []byte) (bool, error) {
	for _, kv := range vp.kvs {
		if bytes.Compare(key, kv.Key) == 0 {
			return true, nil
		}
	}
	return false, fmt.Errorf("key %v does not exist", hex.EncodeToString(key))
}

func (vp *VCPProof) Put(key []byte, value []byte) error {
	for _, kv := range vp.kvs {
		if bytes.Compare(key, kv.Key) == 0 {
			kv.Val = value
			return nil
		}
	}
	vp.kvs = append(vp.kvs, &proofKV{key, value})
	return nil
}
