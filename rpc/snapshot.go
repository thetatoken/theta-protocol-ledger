package rpc

import (
	"bufio"
	"bytes"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/ledger/state"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/rlp"
	"github.com/thetatoken/ukulele/store/treestore"
)

const (
	LatestSnapshot = "theta_snapshot-latest"
)

type snapshotRecord struct {
	K common.Bytes // key
	V common.Bytes // value
	R common.Bytes // account root, if any
}

type snapshotMetadata struct {
	Height uint64
}

type GenSnapshotArgs struct {
}

type GenSnapshotResult struct {
}

func (t *ThetaRPCServer) GenSnapshot(r *http.Request, args *GenSnapshotArgs, result *GenSnapshotResult) (err error) {
	sv, err := t.ledger.GetFinalizedSnapshot()
	if err != nil {
		return err
	}
	currentTime := time.Now().UTC()
	file, err := os.Create("theta_snapshot-" + sv.Hash().String() + "-" + strconv.Itoa(int(sv.Height())) + "-" + currentTime.Format("2006-01-02"))
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)

	metadata := &snapshotMetadata{Height: sv.Height()}
	writeMetadata(writer, metadata)

	db := t.ledger.State().DB()
	sv.GetStore().Traverse(nil, func(k, v common.Bytes) bool {
		writeRecord(writer, k, v, nil)

		if strings.HasPrefix(k.String(), "ls/a/") {
			account := &types.Account{}
			err := types.FromBytes([]byte(v), account)
			if err != nil {
				log.Errorf("Failed to parse account for %v", []byte(v))
				return false
			}
			storage := treestore.NewTreeStore(account.Root, db)
			storage.Traverse(nil, func(ak, av common.Bytes) bool {
				writeRecord(writer, ak, av, account.Root.Bytes())
				return true
			})
			writer.Flush()
			return true
		}
		writer.Flush()
		return true
	})
	return
}

func (t *ThetaRPCServer) LoadSnapshot(r *http.Request, args *GenSnapshotArgs, result *GenSnapshotResult) (err error) {
	file, err := os.Open(LatestSnapshot)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := bufio.NewReader(file)

	metadata, err := readMetadata(reader)
	if err != nil {
		log.Errorf("Failed to load snapshot metadata")
		return err
	}

	db := t.ledger.State().DB()
	store := state.NewStoreView(metadata.Height, common.Hash{}, db)
	var account *types.Account
	accountStorage := treestore.NewTreeStore(common.Hash{}, db)
	for {
		record, err := readRecord(reader)
		if err != nil {
			if err == io.EOF {
				store.Save()
				break
			}
			log.Errorf("Failed to read snapshot record")
			return err
		}
		if len(record.R) == 0 {
			store.Set(record.K, record.V)
		} else {
			if account == nil || !bytes.Equal(account.Root.Bytes(), record.R) {
				root, err := accountStorage.Commit()
				if err != nil {
					log.Errorf("Failed to commit account storage %v", account.Root)
					return err
				}
				if account != nil && bytes.Compare(account.Root.Bytes(), root.Bytes()) != 0 {
					log.Errorf("Account storage root doesn't match %v, %v", account.Root, root)
					panic("Account storage root doesn't match")
				}

				// reset temporary account and account storage
				account = &types.Account{}
				err = types.FromBytes([]byte(record.V), account)
				if err != nil {
					log.Errorf("Failed to parse account for %v", []byte(record.V))
					return err
				}
				accountStorage = treestore.NewTreeStore(common.Hash{}, db)
			}
			accountStorage.Set(record.K, record.V)
		}
	}
	t.ledger.ResetState(metadata.Height, store.Hash())
	return
}

func writeMetadata(writer *bufio.Writer, metadata *snapshotMetadata) {
	raw, err := rlp.EncodeToBytes(metadata)
	if err != nil {
		panic("Failed to encode snapshot metadata")
	}
	// write length first
	_, err = writer.Write(itobs(uint64(len(raw))))
	if err != nil {
		panic("Failed to write snapshot metadata length")
	}
	// write metadata itself
	_, err = writer.Write(raw)
	if err != nil {
		panic("Failed to write snapshot metadata")
	}
}

func readMetadata(reader *bufio.Reader) (*snapshotMetadata, error) {
	metadata := &snapshotMetadata{}
	sizeBytes := make([]byte, 8)
	_, err := reader.Read(sizeBytes)
	if err != nil {
		return metadata, err
	}
	size := bstoi(sizeBytes)
	metadataBytes := make([]byte, size)
	_, err = reader.Read(metadataBytes)
	if err != nil {
		return metadata, err
	}
	err = rlp.DecodeBytes(metadataBytes, metadata)
	return metadata, err
}

func readRecord(reader *bufio.Reader) (*snapshotRecord, error) {
	record := &snapshotRecord{}
	sizeBytes := make([]byte, 8)
	_, err := reader.Read(sizeBytes)
	if err != nil {
		return record, err
	}
	size := bstoi(sizeBytes)
	recordBytes := make([]byte, size)
	_, err = reader.Read(recordBytes)
	if err != nil {
		return record, err
	}
	err = rlp.DecodeBytes(recordBytes, record)
	return record, err
}

func writeRecord(writer *bufio.Writer, k, v, r common.Bytes) {
	raw, err := rlp.EncodeToBytes(snapshotRecord{K: k, V: v, R: r})
	if err != nil {
		panic("Failed to encode storage record")
	}
	// write length first
	_, err = writer.Write(itobs(uint64(len(raw))))
	if err != nil {
		panic("Failed to write storage record length")
	}
	// write record itself
	_, err = writer.Write(raw)
	if err != nil {
		panic("Failed to write storage record")
	}
}

func itobs(val uint64) []byte {
	arr := make([]byte, 8)
	for i := 0; i < 8; i++ {
		arr[i] = byte(val % 10)
		val /= 10
	}
	return arr
}

func bstoi(arr []byte) (val uint64) {
	for i := 0; i < 8; i++ {
		val = val + uint64(arr[i])*uint64(math.Pow10(i))
	}
	return
}
