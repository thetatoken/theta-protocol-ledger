package rpc

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/rlp"
	"github.com/thetatoken/ukulele/store/treestore"
)

const (
	LatestSnapshot = "theta_snapshot-latest"
)

type GenSnapshotArgs struct {
}

type GenSnapshotResult struct {
}

func (t *ThetaRPCServer) GenSnapshot(r *http.Request, args *GenSnapshotArgs, result *GenSnapshotResult) (err error) {
	stub := t.consensus.GetSummary()
	lastFinalizedBlock, err := t.chain.FindBlock(stub.LastFinalizedBlock)
	if err != nil {
		log.Errorf("Failed to get block %v", stub.LastFinalizedBlock)
		return err
	}

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

	if sv.Height() != lastFinalizedBlock.Height {
		return fmt.Errorf("Last finalized block height don't match %v != %v", sv.Height(), lastFinalizedBlock.Height)
	}
	err = writeMetadata(writer, lastFinalizedBlock)
	if err != nil {
		return err
	}

	db := t.ledger.State().DB()
	sv.GetStore().Traverse(nil, func(k, v common.Bytes) bool {
		err = writeRecord(writer, k, v, nil)
		if err != nil {
			panic(err) //TODO replace with return err
		}

		if strings.HasPrefix(k.String(), "ls/a/") {
			account := &types.Account{}
			err := types.FromBytes([]byte(v), account)
			if err != nil {
				log.Errorf("Failed to parse account for %v", []byte(v))
				panic(err)
			}
			storage := treestore.NewTreeStore(account.Root, db)
			storage.Traverse(nil, func(ak, av common.Bytes) bool {
				err = writeRecord(writer, ak, av, account.Root.Bytes())
				if err != nil {
					panic(err)
				}
				return true
			})
		}
		return true
	})
	writer.Flush()
	return
}

func writeMetadata(writer *bufio.Writer, block *core.ExtendedBlock) error {
	raw, err := rlp.EncodeToBytes(*block)
	if err != nil {
		log.Error("Failed to encode snapshot block")
		return err
	}
	// write length first
	_, err = writer.Write(itobs(uint64(len(raw))))
	if err != nil {
		log.Error("Failed to write snapshot block length")
		return err
	}
	// write metadata itself
	_, err = writer.Write(raw)
	if err != nil {
		log.Error("Failed to write snapshot block")
		return err
	}
	return nil
}

func writeRecord(writer *bufio.Writer, k, v, r common.Bytes) error {
	raw, err := rlp.EncodeToBytes(core.SnapshotRecord{K: k, V: v, R: r})
	if err != nil {
		log.Error("Failed to encode storage record")
		return err
	}
	// write length first
	_, err = writer.Write(itobs(uint64(len(raw))))
	if err != nil {
		log.Error("Failed to write storage record length")
		return err
	}
	// write record itself
	_, err = writer.Write(raw)
	if err != nil {
		log.Error("Failed to write storage record")
		return err
	}
	return nil
}

func itobs(val uint64) []byte {
	arr := make([]byte, 8)
	for i := 0; i < 8; i++ {
		arr[i] = byte(val % 10)
		val /= 10
	}
	return arr
}
