package netsync

import (
	"bytes"
	"encoding/hex"

	"github.com/spf13/viper"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/util"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/dispatcher"
	p2ptypes "github.com/thetatoken/ukulele/p2p/types"

	log "github.com/sirupsen/logrus"
)

type RequestManager struct {
	logger *log.Entry

	syncMgr           *SyncManager
	endHashCache      []common.Bytes
	blockRequestCache []common.Bytes
}

func NewRequestManager(syncMgr *SyncManager) *RequestManager {
	rm := &RequestManager{
		syncMgr:           syncMgr,
		endHashCache:      []common.Bytes{},
		blockRequestCache: []common.Bytes{},
	}

	logger := util.GetLoggerForModule("request")
	if viper.GetBool(common.CfgLogPrintSelfID) {
		logger = logger.WithFields(log.Fields{"id": rm.syncMgr.consensus.ID()})
	}
	rm.logger = logger

	return rm
}

func (rm *RequestManager) enqueueBlocks(endHash common.Bytes) {
	for _, b := range rm.endHashCache {
		if bytes.Compare(b, endHash) == 0 {
			rm.logger.WithFields(log.Fields{
				"endHash": endHash,
			}).Debug("Skipping already seen endHash")
			return
		}
	}
	rm.endHashCache = append(rm.endHashCache, endHash)

	tip := rm.syncMgr.consensus.GetTip()
	req := dispatcher.InventoryRequest{ChannelID: common.ChannelIDBlock, Start: tip.Hash.String()}
	// Fixme: since we are broadcasting GetInventory, we might be downloading blocks from multple peers later. Need to fix this.
	rm.logger.WithFields(log.Fields{
		"channelID": req.ChannelID,
		"startHash": req.Start,
		"endHash":   req.End,
	}).Debug("Sending inventory request")
	rm.syncMgr.dispatcher.GetInventory([]string{}, req)
}

func (rm *RequestManager) handleInvRequest(peerID string, req *dispatcher.InventoryRequest) {
	rm.logger.WithFields(log.Fields{
		"channelID": req.ChannelID,
		"startHash": req.Start,
		"endHash":   req.End,
	}).Debug("Received inventory request")
	switch req.ChannelID {
	case common.ChannelIDBlock:
		blocks := []string{}
		if req.Start == "" {
			rm.logger.WithFields(log.Fields{
				"channelID": req.ChannelID,
			}).Error("No start hash is specified in InvRequest")
			return
		}
		curr, err := hex.DecodeString(req.Start)
		if err != nil {
			rm.logger.WithFields(log.Fields{
				"channelID": req.ChannelID,
				"start":     req.Start,
			}).Error("Failed to decode start in InvRequest")
			return
		}
		end, err := hex.DecodeString(req.End)
		if err != nil {
			rm.logger.WithFields(log.Fields{
				"channelID": req.ChannelID,
				"end":       req.End,
			}).Error("Failed to decode end in InvRequest")
			return
		}
		for i := 0; i < dispatcher.MaxInventorySize; i++ {
			blocks = append(blocks, hex.EncodeToString(curr))
			block, err := rm.syncMgr.chain.FindBlock(curr)
			if err != nil {
				rm.logger.WithFields(log.Fields{
					"channelID": req.ChannelID,
					"hash":      curr,
				}).Error("Failed to find block with given hash")
				return
			}
			if len(block.Children) == 0 {
				break
			}

			// Fixme: should we only send blocks on the finalized branch?
			curr = block.Children[0]
			if err != nil {
				rm.logger.WithFields(log.Fields{
					"err":  err,
					"hash": curr,
				}).Error("Failed to load block")
				return
			}
			if bytes.Compare(curr, end) == 0 {
				blocks = append(blocks, hex.EncodeToString(end))
				break
			}
		}
		resp := dispatcher.InventoryResponse{ChannelID: common.ChannelIDBlock, Entries: blocks}
		rm.logger.WithFields(log.Fields{
			"channelID":         resp.ChannelID,
			"len(resp.Entries)": len(resp.Entries),
		}).Debug("Sending inventory response")
		rm.syncMgr.dispatcher.SendInventory([]string{peerID}, resp)
	default:
		rm.logger.WithFields(log.Fields{"channelID": req.ChannelID}).Error("Unsupported channelID in received InvRequest")
	}

}

func (rm *RequestManager) handleInvResponse(peerID string, resp *dispatcher.InventoryResponse) {
	switch resp.ChannelID {
	case common.ChannelIDBlock:
	OUTER:
		for _, hashStr := range resp.Entries {
			hash, err := hex.DecodeString(hashStr)
			if err != nil {
				rm.logger.WithFields(log.Fields{"channelID": resp.ChannelID, "hashStr": hashStr, "err": err}).Error("Failed to parse hash string in InvResponse")
				return
			}
			if _, err := rm.syncMgr.chain.FindBlock(hash); err == nil {
				rm.logger.WithFields(log.Fields{"channelID": resp.ChannelID, "hashStr": hashStr, "err": err}).Warn("Skipping already downloaded hash in InvResponse")
				continue
			}

			for _, b := range rm.blockRequestCache {
				if bytes.Compare(b, hash) == 0 {
					rm.logger.WithFields(log.Fields{
						"endHash": hash,
					}).Debug("Skipping already seen block hash")
					continue OUTER
				}
			}
			rm.endHashCache = append(rm.endHashCache, hash)

			request := dispatcher.DataRequest{
				ChannelID: common.ChannelIDBlock,
				Entries:   []string{hashStr},
			}
			rm.logger.WithFields(log.Fields{
				"channelID":       request.ChannelID,
				"request.Entries": request.Entries,
			}).Debug("Sending data request")
			rm.syncMgr.dispatcher.GetData([]string{peerID}, request)
		}
	default:
		rm.logger.WithFields(log.Fields{"channelID": resp.ChannelID}).Error("Unsupported channelID in received InvRequest")
	}
}

func (rm *RequestManager) handleDataRequest(peerID string, data *dispatcher.DataRequest) {
	switch data.ChannelID {
	case common.ChannelIDBlock:
		for _, hashStr := range data.Entries {
			hash, err := hex.DecodeString(hashStr)
			if err != nil {
				rm.logger.WithFields(log.Fields{"channelID": data.ChannelID, "hashStr": hashStr, "err": err}).Error("Failed to parse hash string in DataRequest")
				return
			}
			block, err := rm.syncMgr.chain.FindBlock(hash)
			if err != nil {
				rm.logger.WithFields(log.Fields{"channelID": data.ChannelID, "hashStr": hashStr, "err": err}).Error("Failed to find hash string locally")
				return
			}
			blockBytes, err := encodeMessage(*(block.Block))
			if err != nil {
				rm.logger.WithFields(log.Fields{"channelID": data.ChannelID, "hashStr": hashStr, "err": err}).Error("Failed to serialize block")
				return
			}
			dataResp := dispatcher.DataResponse{ChannelID: common.ChannelIDBlock, Payload: blockBytes}
			rm.logger.WithFields(log.Fields{
				"channelID": data.ChannelID,
				"hashStr":   hashStr,
			}).Debug("Sending requested block")
			rm.syncMgr.dispatcher.SendData([]string{peerID}, dataResp)
		}
	default:
		rm.logger.WithFields(log.Fields{"channelID": data.ChannelID}).Error("Unsupported channelID in received DataRequest")
	}
}

func (rm *RequestManager) handleDataResponse(peerID string, data *dispatcher.DataResponse) {
	switch data.ChannelID {
	case common.ChannelIDBlock:
		block, err := decodeMessage(data.Payload)
		if err != nil {
			rm.logger.WithFields(log.Fields{
				"error": err,
			}).Error("Failed to decode block")
			return
		}
		msg := &p2ptypes.Message{
			PeerID:  peerID,
			Content: block,
		}
		rm.logger.WithFields(log.Fields{
			"channelID":  data.ChannelID,
			"block.Hash": block.(core.Block).Hash,
		}).Debug("Requested block received")
		rm.syncMgr.AddMessage(msg)
	default:
		rm.logger.WithFields(log.Fields{"channelID": data.ChannelID}).Error("Unsupported channelID in received DataResponse")
	}
}
