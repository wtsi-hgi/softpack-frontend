package environments

import (
	"bytes"
	"encoding/json"
	"errors"
	"sync"

	"github.com/gorilla/websocket"
	"vimagination.zapto.org/jsonrpc"
)

type socket struct {
	*Environments

	mu    sync.RWMutex
	conns map[*websocket.Conn]struct{}
}

type jsonError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type response struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *jsonError      `json:"error,omitempty"`
}

type request struct {
	ID     int    `json:"id"`
	Method string `json:"method"`
	Params any    `json:"params,omitempty"`
}

func (s *socket) ServeConn(conn *websocket.Conn) {
	s.Environments.mu.RLock()
	toSend := encodeBroadcast(s.json)
	s.Environments.mu.RUnlock()

	conn.WriteJSON(toSend)

	s.mu.Lock()
	s.conns[conn] = struct{}{}
	s.mu.Unlock()

	for {
		var request request

		if err := conn.ReadJSON(&request); err != nil {
			break
		}

		// handle request
	}

	s.mu.Lock()
	delete(s.conns, conn)
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
		go conn.WriteJSON(toSend)
	}
	s.mu.RUnlock()
}

var ErrUnknownEndpoint = errors.New("unknown endpoint")
