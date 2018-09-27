package netsync

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/dispatcher"
	"github.com/thetatoken/ukulele/rlp"
)

type MessageIDEnum uint8

const (
	MessageIDInvRequest = iota
	MessageIDInvResponse
	MessageIDDataRequest
	MessageIDDataResponse
)

func encodeMessage(message interface{}) (common.Bytes, error) {
	var buf bytes.Buffer
	var msgID MessageIDEnum
	switch message.(type) {
	case dispatcher.InventoryRequest:
		msgID = MessageIDInvRequest
	case dispatcher.InventoryResponse:
		msgID = MessageIDInvResponse
	case dispatcher.DataRequest:
		msgID = MessageIDDataRequest
	case dispatcher.DataResponse:
		msgID = MessageIDDataResponse
	default:
		return nil, errors.New("Unsupported message type")
	}
	err := rlp.Encode(&buf, msgID)
	if err != nil {
		return nil, err
	}
	err = rlp.Encode(&buf, message)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decodeMessage(raw common.Bytes) (interface{}, error) {
	var msgID MessageIDEnum
	err := rlp.DecodeBytes(raw[:1], &msgID)
	if err != nil {
		return nil, err
	}
	if msgID == MessageIDInvRequest {
		data := dispatcher.InventoryRequest{}
		err = rlp.DecodeBytes(raw[1:], &data)
		return data, err
	} else if msgID == MessageIDInvResponse {
		data := dispatcher.InventoryResponse{}
		err = rlp.DecodeBytes(raw[1:], &data)
		return data, err
	} else if msgID == MessageIDDataRequest {
		data := dispatcher.DataRequest{}
		err = rlp.DecodeBytes(raw[1:], &data)
		return data, err
	} else if msgID == MessageIDDataResponse {
		data := dispatcher.DataResponse{}
		err = rlp.DecodeBytes(raw[1:], &data)
		return data, err
	} else {
		return nil, fmt.Errorf("Unknown message ID: %v", msgID)
	}
}
