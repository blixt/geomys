package geomys

type Event struct {
	Type    string
	Value   interface{}
	stopped bool
}

func NewEvent(event string, value interface{}) *Event {
	return &Event{
		Type:  event,
		Value: value,
	}
}

func (e *Event) Copy() *Event {
	return &Event{
		Type:    e.Type,
		Value:   e.Value,
		stopped: e.stopped,
	}
}

func (e *Event) StopPropagation() {
	e.stopped = true
}
