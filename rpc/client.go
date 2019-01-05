package rpc

import (
	"encoding/json"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/thetatoken/ukulele/rpc/lib/rpc-codec/jsonrpc2"
	"github.com/ybbus/jsonrpc"
	"golang.org/x/net/websocket"
)

type Client interface {
	Call(name string, args []interface{}, result interface{}) error
}

func NewClient(url string) Client {
	if strings.HasSuffix(url, "/ws") {
		return newWSClient(url)
	}
	return newHTTPClient(url)
}

type RPCResponse struct {
	res interface{}
}

func (r *RPCResponse) GetObject(toType interface{}) error {
	js, err := json.Marshal(r.res)
	if err != nil {
		return err
	}

	err = json.Unmarshal(js, toType)
	if err != nil {
		return err
	}

	return nil
}

//
// --------------------- HTTP client -------------------------
//

type HTTPClient struct {
	*jsonrpc.RPCClient
}

func newHTTPClient(url string) HTTPClient {
	return HTTPClient{jsonrpc.NewRPCClient(url)}
}

func (c HTTPClient) Call(name string, args []interface{}, result interface{}) error {
	res, err := c.RPCClient.Call(name, args...)
	if err != nil {
		return err
	}
	return res.GetObject(result)
}

//
// --------------------- WebSocket client -------------------------
//

type WSClient struct {
	*jsonrpc2.Client
	ws *websocket.Conn
}

func newWSClient(url string) WSClient {
	ws, err := websocket.Dial(url, "", url)
	if err != nil {
		log.Fatal(err)
	}
	return WSClient{
		ws:     ws,
		Client: jsonrpc2.NewClient(ws),
	}
}

func (c WSClient) Call(name string, args []interface{}, result interface{}) error {
	return c.Client.Call(name, args, result)
}
