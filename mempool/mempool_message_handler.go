package mempool

import (
	"encoding/hex"
	"fmt"

	"github.com/spf13/viper"

	"github.com/thetatoken/theta/common"
	dp "github.com/thetatoken/theta/dispatcher"
	"github.com/thetatoken/theta/p2p/types"
	"github.com/thetatoken/theta/rlp"
)

//
// MempoolMessageHandler handles the messages received over the
// ChannelIDTransaction channel
//
type MempoolMessageHandler struct {
	mempool *Mempool
}

// CreateMempoolMessageHandler create an instance of the MempoolMessageHandler
func CreateMempoolMessageHandler(mempool *Mempool) *MempoolMessageHandler {
	return &MempoolMessageHandler{
		mempool: mempool,
	}
}

// GetChannelIDs implements the p2p.MessageHandler interface
func (mmh *MempoolMessageHandler) GetChannelIDs() []common.ChannelIDEnum {
	return []common.ChannelIDEnum{
		common.ChannelIDTransaction,
	}
}

// EncodeMessage implements the p2p.MessageHandler interface
func (mmh *MempoolMessageHandler) EncodeMessage(message interface{}) (common.Bytes, error) {
	return rlp.EncodeToBytes(message)
}

func Fuzz(data []byte) int {
	var dataResponse dp.DataResponse
	if err := rlp.DecodeBytes(data, &dataResponse); err != nil {
		return 1
	}
	return 0
}

// ParseMessage implements the p2p.MessageHandler interface
func (mmh *MempoolMessageHandler) ParseMessage(peerID string, channelID common.ChannelIDEnum, rawMessageBytes common.Bytes) (types.Message, error) {
	var dataResponse dp.DataResponse
	rlp.DecodeBytes(rawMessageBytes, &dataResponse)

	rawTx := dataResponse.Payload
	message := types.Message{
		PeerID:    peerID,
		ChannelID: channelID,
		Content:   rawTx,
	}
	return message, nil
}

// HandleMessage implements the p2p.MessageHandler interface
func (mmh *MempoolMessageHandler) HandleMessage(message types.Message) error {
	if message.ChannelID != common.ChannelIDTransaction {
		return fmt.Errorf("Invalid channel for MempoolMessageHandler: %v", message.ChannelID)
	}
	rawTx := message.Content.(common.Bytes)
	logger.Debugf("Received gossiped transaction: %v", hex.EncodeToString(rawTx))

	err := mmh.mempool.InsertTransaction(rawTx)
	if err == DuplicateTxError {
		return nil
	}
	if err != nil {
		return err
	}

	// When using libp2p gossip, we don't need to re-broadcast txs received from other
	// nodes.
	p2pOpt := common.P2POptEnum(viper.GetInt(common.CfgP2POpt))
	if p2pOpt != common.P2POptLibp2p {
		mmh.mempool.BroadcastTx(rawTx)
	}

	return nil
}
