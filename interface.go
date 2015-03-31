package geomys

import (
	"errors"
)

type Handler func(i *Interface, msg interface{}) error

// Default handler.
func defaultHandler(i *Interface, msg interface{}) error {
	return errors.New("There was no handler for the message")
}

type Interface struct {
	Context    interface{}
	channel    chan interface{}
	curHandler int
	handlers   []Handler
	open       bool
}

func NewInterface(context interface{}) *Interface {
	return &Interface{
		Context:    context,
		channel:    make(chan interface{}, 10),
		curHandler: -1,
		handlers:   []Handler{defaultHandler},
		open:       true,
	}
}

func (i *Interface) Close() {
	i.handlers = nil
	i.open = false
	close(i.channel)
}

// Gets a message for the client (or waits until one is available).
func (i *Interface) Get() interface{} {
	return <-i.channel
}

// Handles a message from the client.
func (i *Interface) Handle(msg interface{}) error {
	if !i.open {
		return errors.New("The interface is closed")
	}
	i.curHandler = len(i.handlers) - 1
	err := i.handlers[i.curHandler](i, msg)
	i.curHandler = -1
	return err
}

func (i *Interface) Passthrough(msg interface{}) error {
	if i.curHandler < 1 {
		return errors.New("Cannot pass through")
	}
	i.curHandler--
	return i.handlers[i.curHandler](i, msg)
}

func (i *Interface) PopHandler() {
	if len(i.handlers) < 2 {
		panic("Cannot pop root handler")
	}
	i.handlers[len(i.handlers)-1] = nil
	i.handlers = i.handlers[:len(i.handlers)-1]
}

func (i *Interface) PushHandler(h Handler) {
	i.handlers = append(i.handlers, h)
}

func (i *Interface) ReplaceHandler(h Handler) {
	i.handlers[len(i.handlers)-1] = h
}

// Sends a message to the client.
func (i *Interface) Send(msg interface{}) error {
	if !i.open {
		return errors.New("The interface is closed")
	}
	select {
	case i.channel <- msg:
		return nil
	default:
		i.Close()
		return errors.New("Interface overflowed with messages")
	}
}
