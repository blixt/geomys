# geomys
A super simple framework for writing servers handling many persistent clients.

## Installation

```bash
go get github.com/blixt/geomys
```

## Concepts

### Message

A simple typed data structure (a simple Go struct / JSON object).

### Handler

A function which handles a message coming from the client. The handler has access to the interface and may send a
message back to the client using the `Send` method.

### Interface

An interface between clients and the server. Usually there will be one `Interface` instance per client. The
interface holds a stack of one or more handlers which handle the incoming messages from the client.

The handler that is on the top of the stack will handle incoming messages. It may replace itself (`ReplaceHandler`),
relinquish handling to the previous handler (`PopHandler`) or give control to another handler (`PushHandler`) which
can then choose what to do with its control.

The rationale behind this setup is to let the server move clients between states without having to create one large
state machine. For example, if the client needs to authenticate itself, the server can push an auth handler which
can then pop itself when authentication has been completed.

### Server

A very simple wrapper for a list of `Interface` instances, allowing sending to all interfaces simultaneously and 
cleaning up when an interface has been closed.

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

## Design patterns

Here are some ideas for how to use interfaces and handlers. These examples skip the bootstrapping code for the sake
of brevity.

### Identification

This will require the users to identify themselves before letting them chat with each other. The chat client can
leave out the `Name` field in their `Chat` messages, as it will be replaced with the name provided in the `Ident`
message before being broadcasted.

```go
type Chat struct {
    Name string
    Text string
}

type Ident struct {
    Name string
}

func (e *Example) IdentifyHandler(i *geomys.Interface, msg interface{}) error {
    if ident, ok := msg.(*Ident); ok {
        if ident.Name == "" {
            i.Close()
            return errors.New("Client did not provide a name")
        }
        // Remember the user's ident.
        i.Context = ident
        // Relinquish control to the broadcast handler.
        i.PopHandler()
        return nil
    } else {
        return errors.New("Expected an Ident message")
    }
}

func (e *Example) BroadcastHandler(i *geomys.Interface, msg interface{}) error {
    if chat, ok := msg.(*Chat); ok {
        // Fill in the name in the chat message.
        chat.Name = i.Context.(*Ident).Name
        e.Server.SendAll(chat)
        return nil
    } else {
        return errors.New("Expected a Chat message")
    }
}

func (e *Example) GetInterface(ws *websocket.Conn) *geomys.Interface {
    i := e.Server.NewInterface(nil)
    i.PushHandler(e.BroadcastHandler)
    // This handler will have control first, identifying the user.
    i.PushHandler(e.IdentifyHandler)
    return i
}
```
