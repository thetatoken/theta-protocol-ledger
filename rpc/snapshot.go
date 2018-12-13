package rpc

import (
	"bufio"
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
	f, err := os.Create("theta_snapshot-" + s.Root.String() + "-" + strconv.Itoa(int(s.LastVoteHeight)) + "-" + currentTime.Format("2006-01-02"))
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)

	db := t.ledger.State().DB()
	sv.GetStore().Traverse(nil, func(k, v common.Bytes) bool {
		writeAccount(w, k, v)

		if strings.HasPrefix(k.String(), "ls/a/") {
			account := &types.Account{}
			err := types.FromBytes([]byte(v), account)
			if err != nil {
				log.Errorf("Failed to parse account for %v", []byte(v))
				return false
			}
			storage := treestore.NewTreeStore(account.Root, db)
			storage.Traverse(nil, func(k, v common.Bytes) bool {
				writeAccount(w, k, v)
				return true
			})
			w.Flush()
			return true
		}
		w.Flush()
		return true
	})

	// sess := session.Must(session.NewSession())

	return
}

func writeAccount(w *bufio.Writer, k, v common.Bytes) {
	raw, err := rlp.EncodeToBytes(core.KVPair{Key: k, Value: v})
	if err != nil {
		panic("Failed to encode storage k/v pair")
	}
	_, err = w.Write(raw)
	if err != nil {
		panic("Failed to write storage k/v pair")
	}
}
