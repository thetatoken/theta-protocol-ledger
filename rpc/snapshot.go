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

type GenSnapshotArgs struct {
}

type GenSnapshotResult struct {
}

func (t *ThetaRPCServer) GenSnapshot(r *http.Request, args *GenSnapshotArgs, result *GenSnapshotResult) (err error) {
	sv, err := t.ledger.GetFinalizedSnapshot()
	if err != nil {
		return err
	}
	s := t.consensus.GetSummary()
	currentTime := time.Now().UTC()
	file, err := os.Create("theta_snapshot-" + s.Root.String() + "-" + strconv.Itoa(int(s.LastVoteHeight)) + "-" + currentTime.Format("2006-01-02"))
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)

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
	db := t.ledger.State().DB()
	store := treestore.NewTreeStore(common.Hash{}, db)
	sizeBytes := make([]byte, 4)
	var account *types.Account
	accountStorage := treestore.NewTreeStore(common.Hash{}, db)
	for {
		record, err := readRecord(reader, sizeBytes)
		if err != nil {
			if err == io.EOF {
				_, err := store.Commit()
				if err != nil {
					log.Errorf("Failed to commit store")
					return err
				}
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
	return
}

func readRecord(reader *bufio.Reader, sizeBytes []byte) (*snapshotRecord, error) {
	record := &snapshotRecord{}
	// sizeBytes = sizeBytes[:0]
	sizeBytes = make([]byte, 4)
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
	_, err = writer.Write(itobs(len(raw)))
	if err != nil {
		panic("Failed to write storage record length")
	}
	// write record itself
	_, err = writer.Write(raw)
	if err != nil {
		panic("Failed to write storage record")
	}
}

func itobs(val int) []byte {
	arr := make([]byte, 4) // assume record length won't exceed 9999
	for i := 0; i < 4; i++ {
		arr[i] = byte(val % 10)
		val /= 10
	}
	return arr
}

func bstoi(arr []byte) (val int) {
	for i := 0; i < 4; i++ { // same assumption
		val = val + int(arr[i])*int(math.Pow10(i))
	}
	return
}
