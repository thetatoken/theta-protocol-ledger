package rpc

import (
	"encoding/hex"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/crypto"
)

const txTimeout = 15 * time.Second

type Callback struct {
	txHash   string
	created  time.Time
	Callback func(*core.Block)
}

type TxCallbackManager struct {
	mu               *sync.Mutex
	txHashToCallback map[string]*Callback
	callbacks        []*Callback
}

func NewTxCallbackManager() *TxCallbackManager {
	return &TxCallbackManager{
		mu:               &sync.Mutex{},
		txHashToCallback: make(map[string]*Callback),
		callbacks:        []*Callback{},
	}
}

func (m *TxCallbackManager) AddCallback(txHash common.Hash, cb func(*core.Block)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	txHashStr := txHash.Hex()
	callback := &Callback{
		txHash:   txHashStr,
		created:  time.Now(),
		Callback: cb,
	}
	m.txHashToCallback[txHashStr] = callback
	m.callbacks = append(m.callbacks, callback)
}

func (m *TxCallbackManager) RemoveCallback(txHash common.Hash) (cb *Callback, exists bool) {
	key := txHash.Hex()
	cb, exists = m.txHashToCallback[key]
	if exists {
		delete(m.txHashToCallback, key)
	}
	return
}

func (m *TxCallbackManager) Trim() {
	i := 0
	for ; i < len(m.callbacks); i++ {
		cb := m.callbacks[i]
		if time.Since(cb.created) < txTimeout {
			break
		}
		cb2, ok := m.txHashToCallback[cb.txHash]
		if ok && cb2.created == cb.created {
			delete(m.txHashToCallback, cb.txHash)
		}
	}
	m.callbacks = m.callbacks[i:]
}

var txCallbackManager = NewTxCallbackManager()

func (t *ThetaRPCServer) txCallback() {
	defer t.wg.Done()

	timer := time.NewTicker(1 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-t.ctx.Done():
			return
		case block := <-t.consensus.FinalizedBlocks():
			for _, tx := range block.Txs {
				txHash := crypto.Keccak256Hash(tx)
				cb, ok := txCallbackManager.RemoveCallback(txHash)
				if ok {
					cb.Callback(block)
				}
			}
		case <-timer.C:
			txCallbackManager.Trim()
		}
	}
}

// ------------------------------- BroadcastRawTransaction -----------------------------------

type BroadcastRawTransactionArgs struct {
	TxBytes string `json:"tx_bytes"`
}

type BroadcastRawTransactionResult struct {
	TxHash string            `json:"hash"`
	Block  *core.BlockHeader `json:"block",rlp:"nil"`
}

func (t *ThetaRPCServer) BroadcastRawTransaction(r *http.Request,
	args *BroadcastRawTransactionArgs, result *BroadcastRawTransactionResult) (err error) {
	txBytes, err := hex.DecodeString(args.TxBytes)
	if err != nil {
		return err
	}

	hash := crypto.Keccak256Hash(txBytes)
	result.TxHash = hash.Hex()

	logger.Infof("[rpc] broadcast raw transaction: %v", hex.EncodeToString(txBytes))

	err = t.mempool.InsertTransaction(txBytes)
	if err != nil {
		return err
	}

	finalized := make(chan *core.Block)
	timeout := time.NewTimer(txTimeout)
	defer timeout.Stop()

	txCallbackManager.AddCallback(hash, func(block *core.Block) {
		finalized <- block
	})

	select {
	case block := <-finalized:
		result.Block = block.BlockHeader
		return nil
	case <-timeout.C:
		return errors.New("Timed out waiting for transaction to be included")
	}
}

// ------------------------------- BroadcastRawTransactionAsync -----------------------------------

type BroadcastRawTransactionAsyncArgs struct {
	TxBytes string `json:"tx_bytes"`
}

type BroadcastRawTransactionAsyncResult struct {
	TxHash string `json:"hash"`
}

func (t *ThetaRPCServer) BroadcastRawTransactionAsync(r *http.Request,
	args *BroadcastRawTransactionAsyncArgs, result *BroadcastRawTransactionAsyncResult) (err error) {
	txBytes, err := hex.DecodeString(args.TxBytes)
	if err != nil {
		return err
	}

	hash := crypto.Keccak256Hash(txBytes)
	result.TxHash = hash.Hex()

	logger.Infof("[rpc] broadcast raw transaction: %v", hex.EncodeToString(txBytes))

	return t.mempool.InsertTransaction(txBytes)
}
