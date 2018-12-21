package netsync

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/ledger/state"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/rlp"
	"github.com/thetatoken/ukulele/store/database"
	"github.com/thetatoken/ukulele/store/treestore"
)

func LoadSnapshot(filePath string, db database.Database) (*core.SnapshotMetadata, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := bufio.NewReader(file)

	metadata, err := readMetadata(reader)
	if err != nil {
		log.Errorf("Failed to load snapshot block")
		return nil, err
	}

	store := state.NewStoreView(metadata.Blockheader.Height, common.Hash{}, db)
	var account *types.Account
	accountStorage := treestore.NewTreeStore(common.Hash{}, db)
	for {
		record, err := readRecord(reader)
		if err != nil {
			if err == io.EOF {
				accountStorage.Commit()
				store.Save()
				break
			}
			log.Errorf("Failed to read snapshot record")
			return nil, err
		}
		if len(record.R) == 0 {
			store.Set(record.K, record.V)
		} else {
			if account == nil || !bytes.Equal(account.Root.Bytes(), record.R) {
				root, err := accountStorage.Commit()
				if err != nil {
					log.Errorf("Failed to commit account storage %v", account.Root)
					return nil, err
				}
				if account != nil && bytes.Compare(account.Root.Bytes(), root.Bytes()) != 0 {
					return nil, fmt.Errorf("Account storage root doesn't match %v != %v", account.Root.Bytes(), root.Bytes())
				}

				// reset temporary account and account storage
				account = &types.Account{}
				err = types.FromBytes([]byte(record.V), account)
				if err != nil {
					log.Errorf("Failed to parse account for %v", []byte(record.V))
					return nil, err
				}
				accountStorage = treestore.NewTreeStore(common.Hash{}, db)
			}
			accountStorage.Set(record.K, record.V)
		}
	}
	return metadata, nil
}

func readMetadata(reader *bufio.Reader) (*core.SnapshotMetadata, error) {
	metadata := &core.SnapshotMetadata{}
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

func readRecord(reader *bufio.Reader) (*core.SnapshotRecord, error) {
	record := &core.SnapshotRecord{}
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

func bstoi(arr []byte) (val uint64) {
	for i := 0; i < 8; i++ {
		val = val + uint64(arr[i])*uint64(math.Pow10(i))
	}
	return
}
