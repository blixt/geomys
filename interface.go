package geomys

import (
	"errors"
)

type Handler func(i *Interface, event *Event) error

type Interface struct {
	Context  interface{}
	channel  chan interface{}
	handlers []Handler
	index    int
	open     bool
}

func NewInterface(context interface{}) *Interface {
	return &Interface{
		Context: context,
		channel: make(chan interface{}, 10),
		index:   -1,
		open:    true,
	}
}

func (i *Interface) Close() {
	for _, h := range i.handlers {
		h(i, NewEvent("close", nil))
	}
	i.handlers = nil
	i.open = false
	close(i.channel)
}

// Gets a message for the client (or waits until one is available).
func (i *Interface) Get() interface{} {
	return <-i.channel
}

// Handles an event.
func (i *Interface) Dispatch(event *Event) error {
	if !i.open {
		return errors.New("The interface is closed")
	}
	var err error
	for i.index = len(i.handlers) - 1; i.index >= 0; i.index-- {
		err = i.handlers[i.index](i, event)
		if err != nil || event.stopped {
			break
		}
	}
	i.index = -1
	return err
}

func (i *Interface) PushHandler(h Handler) {
	i.handlers = append(i.handlers, h)
}

// Removes the current handler. Note: This can only be called from a handler.
func (i *Interface) RemoveHandler() {
	if i.index < 0 {
		panic("RemoveHandler can only be called within a handler")
	}
	copy(i.handlers[i.index:], i.handlers[i.index+1:])
	lastIndex := len(i.handlers) - 1
	i.handlers[lastIndex] = nil
	i.handlers = i.handlers[:lastIndex]
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
