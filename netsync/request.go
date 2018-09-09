package netsync

import (
	"bytes"
	"encoding/hex"

	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/dispatcher"
	p2ptypes "github.com/thetatoken/ukulele/p2p/types"
	"github.com/thetatoken/ukulele/rlp"

	log "github.com/sirupsen/logrus"
)

type RequestManager struct {
	syncMgr *SyncManager
}

func NewRequestManager(syncMgr *SyncManager) *RequestManager {
	return &RequestManager{
		syncMgr: syncMgr,
	}
}

func (rm *RequestManager) enqueueBlocks(endHash common.Bytes) {
	tip := rm.syncMgr.consensus.GetTip()
	req := dispatcher.InventoryRequest{ChannelID: common.ChannelIDBlock, Start: tip.Hash.String()}
	// Fixme: since we are broadcasting GetInventory, we might be downloading blocks from multple peers later. Need to fix this.
	rm.syncMgr.dispatcher.GetInventory([]string{}, req)
}

func (rm *RequestManager) handleInvRequest(peerID string, req *dispatcher.InventoryRequest) {
	switch req.ChannelID {
	case common.ChannelIDBlock:
		blocks := []string{}
		if req.Start == "" {
			log.WithFields(log.Fields{"id": rm.syncMgr.consensus.ID(), "channelID": req.ChannelID}).Error("No start hash is specified in InvRequest")
			return
		}
		curr, err := hex.DecodeString(req.Start)
		if err != nil {
			log.WithFields(log.Fields{"id": rm.syncMgr.consensus.ID(), "channelID": req.ChannelID, "start": req.Start}).Error("Failed to decode start in InvRequest")
			return
		}
		end, err := hex.DecodeString(req.End)
		if err != nil {
			log.WithFields(log.Fields{"id": rm.syncMgr.consensus.ID(), "channelID": req.ChannelID, "end": req.End}).Error("Failed to decode end in InvRequest")
			return
		}
		for i := 0; i < dispatcher.MaxInventorySize; i++ {
			blocks = append(blocks, hex.EncodeToString(curr))
			block, err := rm.syncMgr.chain.FindBlock(curr)
			if err != nil {
				log.WithFields(log.Fields{"id": rm.syncMgr.consensus.ID(), "channelID": req.ChannelID, "hash": curr}).Error("Failed to find block with given hash")
				return
			}
			if len(block.Children) == 0 {
				break
			}

			// Fixme: should we only send blocks on the finalized branch?
			curr = block.Children[0]
			if err != nil {
				log.WithFields(log.Fields{"id": rm.syncMgr.consensus.ID(), "err": err, "hash": curr}).Error("Failed to load block")
				return
			}
			if bytes.Compare(curr, end) == 0 {
				blocks = append(blocks, hex.EncodeToString(end))
				break
			}
		}
		resp := dispatcher.InventoryResponse{ChannelID: common.ChannelIDBlock, Entries: blocks}
		rm.syncMgr.dispatcher.SendInventory([]string{peerID}, resp)
	default:
		log.WithFields(log.Fields{"id": rm.syncMgr.consensus.ID(), "channelID": req.ChannelID}).Error("Unsupported channelID in received InvRequest")
	}

}

func (rm *RequestManager) handleInvResponse(peerID string, resp *dispatcher.InventoryResponse) {
	switch resp.ChannelID {
	case common.ChannelIDBlock:
		for _, hashStr := range resp.Entries {
			hash, err := hex.DecodeString(hashStr)
			if err != nil {
				log.WithFields(log.Fields{"id": rm.syncMgr.consensus.ID(), "channelID": resp.ChannelID, "hashStr": hashStr, "err": err}).Error("Failed to parse hash string in InvResponse")
				return
			}
			if _, err := rm.syncMgr.chain.FindBlock(hash); err == nil {
				log.WithFields(log.Fields{"id": rm.syncMgr.consensus.ID(), "channelID": resp.ChannelID, "hashStr": hashStr, "err": err}).Warn("Skipping already downloaded hash in InvResponse")
				continue
			}
			request := dispatcher.DataRequest{
				ChannelID: common.ChannelIDBlock,
				Entries:   []string{hashStr},
			}
			rm.syncMgr.dispatcher.GetData([]string{peerID}, request)
		}
	default:
		log.WithFields(log.Fields{"id": rm.syncMgr.consensus.ID(), "channelID": resp.ChannelID}).Error("Unsupported channelID in received InvRequest")
	}
}

func (rm *RequestManager) handleDataRequest(peerID string, data *dispatcher.DataRequest) {
	switch data.ChannelID {
	case common.ChannelIDBlock:
		for _, hashStr := range data.Entries {
			hash, err := hex.DecodeString(hashStr)
			if err != nil {
				log.WithFields(log.Fields{"id": rm.syncMgr.consensus.ID(), "channelID": data.ChannelID, "hashStr": hashStr, "err": err}).Error("Failed to parse hash string in DataRequest")
				return
			}
			block, err := rm.syncMgr.chain.FindBlock(hash)
			if err != nil {
				log.WithFields(log.Fields{"id": rm.syncMgr.consensus.ID(), "channelID": data.ChannelID, "hashStr": hashStr, "err": err}).Error("Failed to find hash string locally")
				return
			}
			blockBytes, err := rlp.EncodeToBytes(block)
			if err != nil {
				log.WithFields(log.Fields{"id": rm.syncMgr.consensus.ID(), "channelID": data.ChannelID, "hashStr": hashStr, "err": err}).Error("Failed to serialize block")
				return
			}
			dataResp := dispatcher.DataResponse{ChannelID: common.ChannelIDBlock, Payload: blockBytes}
			rm.syncMgr.dispatcher.SendData([]string{peerID}, dataResp)
		}
	default:
		log.WithFields(log.Fields{"id": rm.syncMgr.consensus.ID(), "channelID": data.ChannelID}).Error("Unsupported channelID in received DataRequest")
	}
}

func (rm *RequestManager) handleDataResponse(peerID string, data *dispatcher.DataResponse) {
	switch data.ChannelID {
	case common.ChannelIDBlock:
		block := &blockchain.Block{}
		if err := rlp.DecodeBytes(data.Payload, &block); err != nil {
			log.WithFields(log.Fields{"id": rm.syncMgr.consensus.ID()}).Error("Failed to decode block")
			return
		}
		msg := &p2ptypes.Message{
			PeerID:  peerID,
			Content: block,
		}
		rm.syncMgr.AddMessage(msg)
	default:
		log.WithFields(log.Fields{"id": rm.syncMgr.consensus.ID(), "channelID": data.ChannelID}).Error("Unsupported channelID in received DataResponse")
	}
}
