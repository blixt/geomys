package geomys

import (
	"encoding/json"
	"reflect"

	"github.com/op/go-logging"
	"golang.org/x/net/websocket"
)

var Logger = logging.MustGetLogger("geomys")

// An empty type that implements GetMessageType.
type WebSocketServerBase struct {
}

func (s *WebSocketServerBase) GetMessageType(msg interface{}) (string, error) {
	return reflect.TypeOf(msg).Elem().Name(), nil
}

type WebSocketServer interface {
	GetInterface(ws *websocket.Conn) *Interface
	GetMessage(msgType string) (interface{}, error)
	GetMessageType(msg interface{}) (string, error)
}

func WebSocketHandler(s WebSocketServer) websocket.Handler {
	return func(ws *websocket.Conn) {
		defer disconnectClient(ws)
		Logger.Info("Client connected")

		i := s.GetInterface(ws)
		go receiveToInterface(s, ws, i)
		for {
			if msg := i.Get(); msg != nil {
				Logger.Debug("Sending message %T", msg)
				mustSend(s, ws, msg)
			} else {
				break
			}
		}
	}
}

type intermediate struct {
	Type  string
	Value json.RawMessage
}

func receive(s WebSocketServer, ws *websocket.Conn) (msg interface{}, err error) {
	input := new(intermediate)
	if err = websocket.JSON.Receive(ws, input); err != nil {
		return
	}
	if msg, err = s.GetMessage(input.Type); err != nil {
		msg = nil
		return
	}
	if err = json.Unmarshal(input.Value, msg); err != nil {
		msg = nil
		return
	}
	return
}

func mustReceive(s WebSocketServer, ws *websocket.Conn) interface{} {
	if msg, err := receive(s, ws); err != nil {
		panic(err)
	} else {
		return msg
	}
}

func send(s WebSocketServer, ws *websocket.Conn, msg interface{}) error {
	var (
		msgType string
		msgJSON []byte
		err     error
	)
	if msgType, err = s.GetMessageType(msg); err != nil {
		return err
	}
	if msgJSON, err = json.Marshal(msg); err != nil {
		return err
	}
	if err = websocket.JSON.Send(ws, &intermediate{msgType, msgJSON}); err != nil {
		return err
	}
	return nil
}

func mustSend(s WebSocketServer, ws *websocket.Conn, msg interface{}) {
	if err := send(s, ws, msg); err != nil {
		panic(err)
	}
}

func receiveToInterface(s WebSocketServer, ws *websocket.Conn, i *Interface) {
	for {
		if msg, err := receive(s, ws); err != nil {
			Logger.Debug("Stopping receive: %s", err)
			i.Close()
			break
		} else {
			Logger.Debug("Received message %T", msg)
			if err := i.Handle(msg); err != nil {
				Logger.Warning("Client caused error: %s", err)
			}
		}
	}
}

func disconnectClient(ws *websocket.Conn) {
	if r := recover(); r != nil {
		Logger.Error("Client error: %s", r)
	}
	if err := ws.Close(); err != nil {
		Logger.Error("Failed to close web socket (%s)", err)
	}
	Logger.Info("Client disconnected")
}
