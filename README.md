# geomys
A super simple framework for writing servers handling many persistent clients.

## Installation

```bash
go get github.com/blixt/geomys
```

## WebSocket support

This library is built with the web in mind, so it makes WebSocket connections easier. It uses a simple protocol for
translating Go structs to JSON objects and back. Here's what a WebSocket message looks like in Go and as JSON:

**Go:** `&MyDataType{Hello: "World", Answer: 42}`

**JSON:** `{"Type": "MyDataType", "Value": {"Hello": "World", "Answer": 42}}`

In order to convert the data back to Go, *geomys* needs to know which struct type to unmarshal the JSON into, which
it figures out by calling `GetMessage("MyDataType")` on your `WebSocketServer` implementation, expecting an empty
struct back which the `Value` field will be unmarshaled into.

## Example

This example shows how to listen for web sockets and broadcasting their `Chat` messages to all other connected
sockets.

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/blixt/geomys"
	"golang.org/x/net/websocket"
)

type Acknowledgement struct {
}

type Chat struct {
	Text string
}

// The example implementation of the server.
type Example struct {
	geomys.WebSocketServerBase
	Server *geomys.Server
}

func NewExample() *Example {
	return &Example{Server: geomys.NewServer()}
}

// A handler which we'll add to all incoming clients.
func (e *Example) BroadcastHandler(i *geomys.Interface, msg interface{}) error {
	// Broadcast the message to all interfaces.
	e.Server.SendAll(msg)
	// Acknowledge that the message was received.
	i.Send(&Acknowledgement{})
	return nil
}

// Handles an incoming socket by returning an interface to the server.
func (e *Example) GetInterface(ws *websocket.Conn) *geomys.Interface {
	i := e.Server.NewInterface(nil)
	// Handle all connections with our broadcast handler.
	i.PushHandler(e.BroadcastHandler)
	return i
}

// Returns an empty struct for the provided type name.
func (e *Example) GetMessage(msgType string) (interface{}, error) {
	switch msgType {
	case "Chat":
		return new(Chat), nil
	default:
		return nil, fmt.Errorf("Unsupported message type %s", msgType)
	}
}

func main() {
	// Our Example type implements geomys.WebSocketServer.
	example := NewExample()

	// Start the WebSocket server.
	fmt.Println("Starting server on port 1337...")
	http.Handle("/socket", geomys.WebSocketHandler(example))
	http.ListenAndServe(":1337", nil)
}
```
