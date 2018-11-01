package blockchain

import (
	log "github.com/sirupsen/logrus"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/crypto"
	"github.com/thetatoken/ukulele/store"
)

// txIndexKey constructs the DB key for the given transaction hash.
func txIndexKey(hash common.Hash) common.Bytes {
	return append(common.Bytes("tx/"), hash[:]...)
}

// TxIndexEntry is a positional metadata to help looking up a transaction given only its hash.
type TxIndexEntry struct {
	BlockHash   common.Hash
	BlockHeight uint64
	Index       uint64
}

// addTxsToIndex adds transactions in given block to index.
func (ch *Chain) addTxsToIndex(block *core.ExtendedBlock) {
	for idx, tx := range block.Txs {
		txIndexEntry := TxIndexEntry{
			BlockHash:   common.BytesToHash(block.Hash),
			BlockHeight: block.Height,
			Index:       uint64(idx),
		}
		txHash := crypto.Keccak256Hash(tx)
		key := txIndexKey(txHash)
		err := ch.store.Put(key, txIndexEntry)
		if err != nil {
			log.Panic(err)
		}
	}
}

// FindTxByHash looks up transaction by hash and additionaly returns the containing block.
func (ch *Chain) FindTxByHash(hash common.Hash) (tx common.Bytes, block *core.ExtendedBlock, founded bool) {
	txIndexEntry := &TxIndexEntry{}
	err := ch.store.Get(txIndexKey(hash), txIndexEntry)
	if err != nil {
		if err == store.ErrKeyNotFound {
			return nil, nil, false
		}
		log.Panic(err)
	}
	block, err = ch.FindBlock(txIndexEntry.BlockHash[:])
	if err != nil {
		if err == store.ErrKeyNotFound {
			return nil, nil, false
		}
		log.Panic(err)
	}
	return block.Txs[txIndexEntry.Index], block, true
}
