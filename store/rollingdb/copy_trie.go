package rollingdb

import (
	"bytes"
	"log"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/ledger/state"
	"github.com/thetatoken/theta/ledger/types"
	"github.com/thetatoken/theta/store/database"
	"github.com/thetatoken/theta/store/trie"
)

func copyState(source database.Database, target database.Batch, root common.Hash) {
	copyTrie(source, target, root)

	sv := state.NewStoreView(0, root, source)

	sv.GetStore().Traverse(nil, func(k, v common.Bytes) bool {
		if bytes.HasPrefix(k, []byte("ls/a")) {
			account := &types.Account{}
			err := types.FromBytes([]byte(v), account)
			if err != nil {
				logger.Errorf("Failed to parse account for %v", []byte(v))
				panic(err)
			}
			if account.Root != (common.Hash{}) {
				copyTrie(source, target, account.Root)
			}
		}
		return true
	})
}

func copyTrie(source database.Database, target database.Batch, root common.Hash) {
	tr, err := trie.New(root, trie.NewDatabase(source))
	if err != nil {
		logger.Panic(err)
	}
	it := tr.NodeIterator(nil)
	for it.Next(true) {
		if it.Hash() != (common.Hash{}) {
			hash := it.Hash()
			val, err := source.Get(hash.Bytes())
			if err != nil {
				log.Panic(err)
			}
			err = target.Put(hash.Bytes(), val)
			if err != nil {
				logger.Panic(err)
			}

			if target.ValueSize() > database.IdealBatchSize {
				if err := target.Write(); err != nil {
					logger.Panicf("Failed to copy trie: %v", err)
				}
				target.Reset()
			}

		}
	}
	if err := target.Write(); err != nil {
		logger.Panicf("Failed to copy trie: %v", err)
	}
}
