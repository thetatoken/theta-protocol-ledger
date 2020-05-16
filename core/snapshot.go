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

const SnapshotHeaderMagic = "ThetaToDaMoon"
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
	Header *BlockHeader
	Proof  VCPProof
}

type SnapshotSecondBlock struct {
	Header *BlockHeader
}

type SnapshotThirdBlock struct {
	Header  *BlockHeader
	VoteSet *VoteSet
}

type SnapshotBlockTrio struct {
	First  SnapshotFirstBlock
	Second SnapshotSecondBlock
	Third  SnapshotThirdBlock
}

type SnapshotHeader struct {
	Magic   string
	Version uint
}

type SnapshotMetadata struct {
	ProofTrios []SnapshotBlockTrio
	TailTrio   SnapshotBlockTrio
}

type LastCheckpoint struct {
	CheckpointHeader    *BlockHeader
	IntermediateHeaders []*BlockHeader
}

func WriteSnapshotHeader(writer *bufio.Writer, snapshotHeader *SnapshotHeader) error {
	raw, err := rlp.EncodeToBytes(*snapshotHeader)
	if err != nil {
		logger.Errorf("Failed to encode snapshot header: %v", err)
		return err
	}
	err = writeBytes(writer, raw)
	return err
}

func WriteLastCheckpoint(writer *bufio.Writer, lastCheckpoint *LastCheckpoint) error {
	raw, err := rlp.EncodeToBytes(*lastCheckpoint)
	if err != nil {
		logger.Errorf("Failed to encode last checkpoint: %v", err)
		return err
	}
	err = writeBytes(writer, raw)
	return err
}

func WriteMetadata(writer *bufio.Writer, metadata *SnapshotMetadata) error {
	raw, err := rlp.EncodeToBytes(*metadata)
	if err != nil {
		logger.Errorf("Failed to encode metadata: %v", err)
		return err
	}
	err = writeBytes(writer, raw)
	return err
}

func WriteRecord(writer *bufio.Writer, k, v common.Bytes) error {
	record := SnapshotTrieRecord{K: k, V: v}
	raw, err := rlp.EncodeToBytes(record)
	if err != nil {
		logger.Errorf("Failed to encode record: %v", err)
		return err
	}
	err = writeBytes(writer, raw)
	return err
}

func writeBytes(writer *bufio.Writer, raw []byte) error {
	// write length first
	_, err := writer.Write(Itobytes(uint64(len(raw))))
	if err != nil {
		logger.Errorf("Failed to write snapshot object length: %v", err)
		return err
	}
	// write the object itself
	_, err = writer.Write(raw)
	if err != nil {
		logger.Errorf("Failed to write snapshot object: %v", err)
		return err
	}
	err = writer.Flush()
	if err != nil {
		return fmt.Errorf("Failed to flush snapshot object, %v", err)
	}
	return nil
}

func ReadRecord(file *os.File, obj interface{}) (uint64, error) {
	sizeBytes := make([]byte, 8)
	n, err := io.ReadAtLeast(file, sizeBytes, 8)
	if err != nil {
		return 0, err
	}
	if n < 8 {
		return 0, fmt.Errorf("Failed to read record length")
	}
	size := Bytestoi(sizeBytes)
	bytes := make([]byte, size)
	n, err = io.ReadAtLeast(file, bytes, int(size))
	if err != nil {
		return 0, err
	}
	if uint64(n) < size {
		return 0, fmt.Errorf("Failed to read record, %v < %v", n, size)
	}
	err = rlp.DecodeBytes(bytes, obj)
	return size, err
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
