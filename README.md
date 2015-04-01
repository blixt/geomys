# geomys
A simple framework for writing servers handling many persistent clients.

## Installation

```bash
go get github.com/blixt/geomys
```

## Concepts

### Message

A simple typed data structure (a simple Go struct / JSON object).

### Interface

An interface between clients and the server. Usually there will be one `Interface` instance per client. The
interface holds a stack of one or more handlers which handle events such as incoming messages from the client. The
handlers may send messages back to the client using the interface's `Send` method.

The interface also has a `Context` field which can be set to any value. This can for example be set to a session
object representing the user that the interface communicates with.

### Handler

A function which handles an event, such as a message coming from the client. The handler has access to the interface
and may send a message to the client using the `Send` method or dispatch more events with the `Dispatch` method.

### Event

An event that can be handled by a handler. The event will bubble through the handlers unless stopped by calling the
`StopPropagation` method or returning an error in a handler. By default there is only the `"message"` event which is
dispatched whenever a client sends a message.

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

// Our simple handler for all connected clients.
func (e *Example) BroadcastHandler(i *geomys.Interface, event *geomys.Event) error {
	if event.Type == "message" {
		// Broadcast the message to all interfaces.
		e.Server.SendAll(event.Value)
		// Acknowledge that the message was received.
		i.Send(&Acknowledgement{})
	}
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
	example := &Example{Server: geomys.NewServer()}

	// Start the WebSocket server.
	fmt.Println("Starting server on port 1337...")
	// Example implements geomys.WebSocketServer which makes web sockets easy.
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

func (e *Example) IdentifyHandler(i *geomys.Interface, event *geomys.Event) error {
    switch msg := event.Value.(type) {
    case *Ident:
        if msg.Name == "" {
            i.Close()
            return errors.New("Client did not provide a name")
        }
        // Remember the user's ident.
        i.Context = msg
        // Stop looking for ident messages.
        i.RemoveHandler()
	// Prevent this event from bubbling to the broadcast handler.
	event.StopPropagation()
    default:
        return errors.New("Expected an Ident message")
    }
    return nil
}

func (e *Example) BroadcastHandler(i *geomys.Interface, event *geomys.Event) error {
    switch msg := event.Value.(type) {
    case *Chat:
        // Fill in the name in the chat message.
        msg.Name = i.Context.(*Ident).Name
        e.Server.SendAll(msg)
    default:
        return errors.New("Expected a Chat message")
    }
    return nil
}

func (e *Example) GetInterface(ws *websocket.Conn) *geomys.Interface {
    i := e.Server.NewInterface(nil)
    i.PushHandler(e.BroadcastHandler)
    // This handler will have control first, identifying the user.
    i.PushHandler(e.IdentifyHandler)
    return i
}
```
