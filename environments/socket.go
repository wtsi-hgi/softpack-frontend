package environments

import (
	"bytes"
	"encoding/json"
	"errors"
	"sync"

	"golang.org/x/net/websocket"
	"vimagination.zapto.org/jsonrpc"
)

type socket struct {
	*Environments

	mu    sync.RWMutex
	conns map[*conn]struct{}
}

func (s *socket) ServeConn(wconn *websocket.Conn) {
	var c conn

	s.Environments.mu.RLock()
	toSend := encodeBroadcast(s.json)
	s.Environments.mu.RUnlock()

	s.mu.Lock()
	c.rpc = jsonrpc.New(wconn, &c)
	s.conns[&c] = struct{}{}
	s.mu.Unlock()

	c.rpc.SendData(toSend)

	c.rpc.Handle()

	s.mu.Lock()
	delete(s.conns, &c)
	s.mu.Unlock()
}

func encodeBroadcast(data any) json.RawMessage {
	var buf bytes.Buffer

	json.NewEncoder(&buf).Encode(jsonrpc.Response{
		ID:     -1,
		Result: data,
	})

	return json.RawMessage(buf.Bytes())
}

func (s *socket) SendToAll(data any) {
	toSend := encodeBroadcast(data)

	s.mu.RLock()
	for conn := range s.conns {
		go conn.rpc.SendData(toSend)
	}
	s.mu.RUnlock()
}

type conn struct {
	rpc *jsonrpc.Server
}

func (conn) HandleRPC(method string, data json.RawMessage) (any, error) {
	return nil, ErrUnknownEndpoint
}

var ErrUnknownEndpoint = errors.New("unknown endpoint")
