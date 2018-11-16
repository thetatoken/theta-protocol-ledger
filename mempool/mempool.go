package mempool

import (
	"context"
	"encoding/hex"
	"errors"
	"math/big"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/clist"
	"github.com/thetatoken/ukulele/common/math"
	"github.com/thetatoken/ukulele/common/pqueue"
	"github.com/thetatoken/ukulele/core"
	dp "github.com/thetatoken/ukulele/dispatcher"
)

type MempoolError string

func (m MempoolError) Error() string {
	return string(m)
}

const DuplicateTxError = MempoolError("Transaction already seen")

//
// mempoolTransaction implements the pqueue.Element interface
//
type mempoolTransaction struct {
	index          int
	rawTransaction common.Bytes
	feeAmount      *big.Int
}

var _ pqueue.Element = (*mempoolTransaction)(nil)

func (mt *mempoolTransaction) Priority() *big.Int {
	return mt.feeAmount
}

func (mt *mempoolTransaction) SetIndex(index int) {
	mt.index = index
}

func (mt *mempoolTransaction) GetIndex() int {
	return mt.index
}

func createMempoolTransaction(rawTransaction common.Bytes, feeAmount *big.Int) *mempoolTransaction {
	return &mempoolTransaction{
		rawTransaction: rawTransaction,
		feeAmount:      feeAmount,
	}
}

//
// Mempool manages the transactions submitted by the clients
// or relayed from peers
//
type Mempool struct {
	mutex *sync.Mutex

	ledger     core.Ledger
	dispatcher *dp.Dispatcher

	newTxs       *clist.CList          // new transactions, to be gossiped to other nodes
	candidateTxs *pqueue.PriorityQueue // candidate transactions for new block assembly, ordered by the transaction fee (high to low)
	txBookeepper transactionBookkeeper

	// Life cycle
	wg      *sync.WaitGroup
	quit    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
}

// CreateMempool creates an instance of Mempool
func CreateMempool(dispatcher *dp.Dispatcher) *Mempool {
	return &Mempool{
		mutex:        &sync.Mutex{},
		dispatcher:   dispatcher,
		newTxs:       clist.New(),
		candidateTxs: pqueue.CreatePriorityQueue(),
		txBookeepper: createTransactionBookkeeper(defaultMaxNumTxs),
		wg:           &sync.WaitGroup{},
	}
}

// SetLedger sets the ledger for the mempool
func (mp *Mempool) SetLedger(ledger core.Ledger) {
	mp.ledger = ledger
}

// InsertTransaction inserts the incoming transaction to mempool (submitted by the clients or relayed from peers)
func (mp *Mempool) InsertTransaction(rawTx common.Bytes) error {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	if mp.txBookeepper.hasSeen(rawTx) {
		log.Infof("[mempool] Transaction already seen: %v", hex.EncodeToString(rawTx))
		return DuplicateTxError
	}

	feeAmount, checkTxRes := mp.ledger.ScreenTx(rawTx)
	if !checkTxRes.IsOK() {
		return errors.New(checkTxRes.Message)
	}

	log.Debugf("[mempool] Insert tx: %v, fee: %v", hex.EncodeToString(rawTx), feeAmount)

	// only record the transactions that passed the screening. This is because that
	// an invalid transaction could becoume valid later on. For example, assume expected
	// sequence for an account is 6. The account accidently submits txA (seq = 7), got rejected.
	// He then submit txB(seq = 6), and then txA(seq = 7) again. For the second submission, txA
	// should not be rejected even though it has been submitted earlier.
	mp.txBookeepper.record(rawTx)

	mptx := createMempoolTransaction(rawTx, feeAmount)
	mp.newTxs.PushBack(mptx)
	mp.candidateTxs.Push(mptx)

	return nil
}

// Start needs to be called when the Mempool starts
func (mp *Mempool) Start(ctx context.Context) error {
	c, cancel := context.WithCancel(ctx)
	mp.ctx = c
	mp.cancel = cancel

	mp.wg.Add(1)
	go mp.broadcastTransactionsRoutine()

	return nil
}

// Stop needs to be called when the Mempool stops
func (mp *Mempool) Stop() {
	mp.cancel()
}

// Wait suspends the caller goroutine
func (mp *Mempool) Wait() {
	mp.wg.Wait()
}

// Lock is for the caller to lock/unlock the Mempool and perform safely update
func (mp *Mempool) Lock() {
	mp.mutex.Lock()
}

// Unlock is for the caller to lock/unlock the Mempool and perform safely update
func (mp *Mempool) Unlock() {
	mp.mutex.Unlock()
}

// Size returns the number of transactions in the Mempool
func (mp *Mempool) Size() int {
	return mp.candidateTxs.NumElements()
}

// Reap returns a list of valid raw transactions and remove these
// transactions from the candidate pool. maxNumTxs == 0 means
// none, maxNumTxs < 0 means uncapped. Note that Reap does NOT remove
// the transactions from the candidateTxs list. Instead, the consensus engine needs
// to call the Mempool.Update() function to remove the committed transactions
// RUNTIME COMPLEXITY: k*log(n), where k is the number transactions to reap,
// and n is the number of transactions in the candidate pool
func (mp *Mempool) Reap(maxNumTxs int) []common.Bytes {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	if maxNumTxs == 0 {
		return []common.Bytes{}
	} else if maxNumTxs < 0 {
		maxNumTxs = mp.candidateTxs.NumElements()
	} else {
		maxNumTxs = math.MinInt(mp.candidateTxs.NumElements(), maxNumTxs)
	}

	txs := make([]common.Bytes, 0, maxNumTxs)
	for i := 0; i < maxNumTxs; i++ {
		if mp.candidateTxs.IsEmpty() {
			break
		}
		mptx := mp.candidateTxs.Pop().(*mempoolTransaction)
		txs = append(txs, mptx.rawTransaction)

		log.Debugf("[mempool] Reap tx: %v, fee: %v",
			hex.EncodeToString(mptx.rawTransaction), mptx.feeAmount)
	}

	return txs
}

// Update removes the committed transactions from the transaction candidate list
// RUNTIME COMPLEXITY: k*log(n), where k is the number committed raw transactions,
// and n is the number of transactions in the candidate pool
func (mp *Mempool) Update(committedRawTxs []common.Bytes) bool {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	committedRawTxMap := make(map[string]bool)
	for _, rawtx := range committedRawTxs {
		committedRawTxMap[string(rawtx)] = true
	}

	elementList := mp.candidateTxs.ElementList()
	elemsTobeRemoved := []pqueue.Element{}
	for _, elem := range *elementList {
		mptx := elem.(*mempoolTransaction)
		rawTx := mptx.rawTransaction
		if _, exists := committedRawTxMap[string(rawTx[:])]; exists {
			elemsTobeRemoved = append(elemsTobeRemoved, elem)
			log.Debugf("[mempool] tx to be removed: %v, fee: %v", hex.EncodeToString(rawTx), mptx.feeAmount)
		}
	}

	// Note after each iteration, the indices of the elems could change
	// So we need elem.GetIndex() to return the updated index
	for _, elem := range elemsTobeRemoved {
		mp.candidateTxs.Remove(elem.GetIndex())
	}

	return true
}

// Flush removes all transactions from the Mempool and the transactionBookkeeper
func (mp *Mempool) Flush() {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	mp.txBookeepper.reset()

	for !mp.candidateTxs.IsEmpty() {
		mp.candidateTxs.Pop()
	}
}

// broadcastTransactionRoutine broadcasts transactions to neighoring peers
func (mp *Mempool) broadcastTransactionsRoutine() {
	defer mp.wg.Done()

	var next *clist.CElement
	for {
		select {
		case <-mp.ctx.Done():
			mp.stopped = true
			return
		default:
		}

		if next == nil {
			next = mp.newTxs.FrontWait() // Wait until a tx is available
		}

		mptx := next.Value.(*mempoolTransaction)

		// Broadcast the transaction
		data := dp.DataResponse{
			ChannelID: common.ChannelIDTransaction,
			Checksum:  []byte(""), // TODO: calculate the checksum
			Payload:   mptx.rawTransaction,
		}

		peerIDs := []string{} // empty peerID list means broadcasting to all neighboring peers
		mp.dispatcher.SendData(peerIDs, data)

		curr := next
		next = curr.NextWait()
		mp.newTxs.Remove(curr) // already broadcasted, should remove
	}
}
