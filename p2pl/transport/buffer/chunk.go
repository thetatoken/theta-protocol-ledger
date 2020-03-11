package buffer

import (
	"bytes"
	"encoding/binary"
	"fmt"

	log "github.com/sirupsen/logrus"
	cmn "github.com/thetatoken/theta/p2pl/common"
)

const (
	headerSize          = 16
	seqIDOffset         = 0
	payloadSizeOffset   = 4
	isEOFOffset         = 8
	payloadOffset       = headerSize
	maxChunkPayloadSize = cmn.MaxChunkSize - headerSize
	markerNotEOF        = byte(0x00)
	markerEOF           = byte(0x01)
	chunkTypePing       = byte(0x01)
	chunkTypePong       = byte(0x02)
	chunkTypeMsg        = byte(0x03)
)

/*
Chunk Format:

  bytes[0..3]  : seqID, 32 bits (int32)
  bytes[4..7]  : payloadSize, 32 bits (int32)
  bytes[8]     : isEOF, 8 bits
  bytes[9..15] : reserved
  bytes[16..payloadSize+15]: the actual payload

                      Chunk Bit Map
0                   1                   2                   3
0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                         seqID (32 bits)                       |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                      payloadSize (32 bits)                    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|    isEOF      |                    reserved                   |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                           reserved                            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
|                           payload                             |
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
*/
type Chunk struct {
	bytes []byte
}

// NewChunk will create a chunk from the given "content" byte slice. It extracts
// the subslice content[startIdx:startIdx+payloadSize] as the payload of the chunk,
// and adds the meta data (isEOF, seqID) after the payload
func NewChunk(content []byte, startIdx, payloadSize int32, isEOF byte, seqID int32) *Chunk {
	//logger.Debugf("NewChunk: startIdx = %v, payloadSize = %v, isEOF = %v, seqID = %v", startIdx, payloadSize, isEOF, seqID)

	numBytes := headerSize + payloadSize
	bytes := make([]byte, numBytes)

	copy(bytes[seqIDOffset:seqIDOffset+4], int32ToBytes(seqID))
	copy(bytes[payloadSizeOffset:payloadSizeOffset+4], int32ToBytes(payloadSize))
	bytes[isEOFOffset] = isEOF
	copy(bytes[payloadOffset:payloadOffset+payloadSize], content[startIdx:startIdx+payloadSize])

	return &Chunk{
		bytes: bytes,
	}
}

func NewUninitializedChunk(payloadSize int32) *Chunk {
	numBytes := headerSize + payloadSize
	bytes := make([]byte, numBytes)
	return &Chunk{
		bytes: bytes,
	}
}

func NewEmptyChunk(seqID int32) *Chunk {
	numBytes := headerSize
	bytes := make([]byte, numBytes)
	copy(bytes[seqIDOffset:seqIDOffset+4], int32ToBytes(seqID))
	copy(bytes[payloadSizeOffset:payloadSizeOffset+4], int32ToBytes(0))
	bytes[isEOFOffset] = markerEOF // EOF
	return &Chunk{
		bytes: bytes,
	}
}

func NewChunkFromRawBytes(bytes []byte) (*Chunk, error) {
	if len(bytes) < headerSize {
		return nil, fmt.Errorf("At least %v bytes needed to create a chunk", headerSize)
	}

	chunk := &Chunk{
		bytes: bytes,
	}

	err := chunk.sanityCheck()
	if err != nil {
		return nil, err
	}

	return chunk, nil
}

func (chunk Chunk) String() string {
	return fmt.Sprintf("Chunk{%X EOF:%v}", chunk.bytes, chunk.IsEOF())
}

func (chunk *Chunk) Bytes() []byte {
	return chunk.bytes
}

func (chunk *Chunk) IsEmpty() bool {
	return (chunk.bytes == nil || len(chunk.bytes) <= headerSize)
}

func (chunk *Chunk) IsValid() bool {
	return chunk.sanityCheck() != nil
}

func (chunk *Chunk) SeqID() int32 {
	seqIDBytes := chunk.bytes[seqIDOffset : seqIDOffset+4]
	seqID := int32FromBytes(seqIDBytes)
	return seqID
}

func (chunk *Chunk) payloadSize() int32 {
	payloadSizeBytes := chunk.bytes[payloadSizeOffset : payloadSizeOffset+4]
	payloadSize := int32FromBytes(payloadSizeBytes)
	return payloadSize
}

func (chunk *Chunk) IsEOF() bool {
	return chunk.bytes[isEOFOffset] == markerEOF
}

func (chunk *Chunk) Payload() []byte {
	return chunk.bytes[payloadOffset:]
}

func (chunk *Chunk) sanityCheck() error {
	numBytes := int32(len(chunk.bytes))
	payloadSize := chunk.payloadSize()
	exptedMinNumBytes := payloadSize + headerSize
	if numBytes < exptedMinNumBytes {
		errMsg := fmt.Sprintf("Invalid chunk, numBytes = %v, exptedMinNumBytes = %v", numBytes, exptedMinNumBytes)
		log.Errorf(errMsg)
		return fmt.Errorf(errMsg)
	}

	seqID := chunk.SeqID()
	if seqID < 0 {
		errMsg := fmt.Sprintf("Invalid chunk seqID %v", seqID)
		log.Errorf(errMsg)
		return fmt.Errorf(errMsg)
	}

	return nil
}

func int32FromBytes(data []byte) (val int32) {
	buf := bytes.NewBuffer(data)
	binary.Read(buf, binary.BigEndian, &val)
	return val
}

func int32ToBytes(val int32) (data []byte) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, val)
	return buf.Bytes()
}
