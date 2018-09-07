package mempool

import "github.com/thetatoken/ukulele/common"

type mempoolTransaction struct {
	rawTransaction common.Bytes
}

type Mempool struct {
}

func (mp *Mempool) CheckTransaction(mptx *mempoolTransaction) error {
	// TODO: to be implemented..
	return nil
}

func (mp *Mempool) OnStart() error {
	go mp.broadcastTransactionRoutine()
	return nil
}

func (mp *Mempool) OnStop() {
}

func (mp *Mempool) broadcastTransactionRoutine() {

}
