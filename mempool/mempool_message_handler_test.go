package mempool

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/common"
	dp "github.com/thetatoken/theta/dispatcher"
	p2psim "github.com/thetatoken/theta/p2p/simulation"
	"github.com/thetatoken/theta/rlp"
)

func TestMempoolMessageHandler(t *testing.T) {
	assert := assert.New(t)

	p2psimnet := p2psim.NewSimnetWithHandler(nil)
	mempool, _ := newTestMempool("peer0", p2psimnet)

	// Simulate receiving a transaction gossip message from the p2p network
	txReceivedChan := make(chan bool)
	go func() {
		mmh := CreateMempoolMessageHandler(mempool)

		tx1 := createTestRawTx("tx1")
		tx2 := createTestRawTx("tx2")
		tx3 := createTestRawTx("tx3")
		txs := []common.Bytes{tx1, tx2, tx3}

		for _, rawTx := range txs {
			dataResponse := dp.DataResponse{
				ChannelID: common.ChannelIDTransaction,
				Payload:   rawTx,
			}
			contentBytes, err := rlp.EncodeToBytes(dataResponse)
			if err != nil {
				log.Errorf("Error encoding: %v, err: %v", dataResponse, err)
				return
			}

			message, err := mmh.ParseMessage("peer1", common.ChannelIDTransaction, contentBytes)
			if err != nil {
				log.Errorf("Error parsing raw message: %v, err: %v", contentBytes, err)
				return
			}

			err = mmh.HandleMessage(message)
			if err != nil {
				log.Errorf("[p2p] Error handling message: %v, err: %v", message, err)
				return
			}
		}

		txReceivedChan <- true
	}()

	txReceived := <-txReceivedChan
	assert.True(txReceived)

	assert.Equal(3, mempool.Size())

	log.Infof("----- Reap all transactions -----")
	reapedRawTxs := mempool.Reap(-1)
	assert.Equal(3, len(reapedRawTxs))
	log.Infof("reapedRawTxs[0]: %v", string(reapedRawTxs[0]))
	log.Infof("reapedRawTxs[1]: %v", string(reapedRawTxs[1]))
	log.Infof("reapedRawTxs[2]: %v", string(reapedRawTxs[2]))
	assert.Equal("tx2", string(reapedRawTxs[0][:]))
	assert.Equal("tx1", string(reapedRawTxs[1][:]))
	assert.Equal("tx3", string(reapedRawTxs[2][:]))
}
