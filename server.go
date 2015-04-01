package geomys

type Server struct {
	Interfaces []*Interface
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) NewInterface(context interface{}) *Interface {
	i := NewInterface(context)
	s.Interfaces = append(s.Interfaces, i)
	return i
}

func (s *Server) DispatchAll(event *Event) {
	for _, i := range s.Interfaces {
		i.Dispatch(event.Copy())
	}
}

func (s *Server) SendAll(msg interface{}) {
	count, deleted := len(s.Interfaces), 0
	for index, i := range s.Interfaces {
		if err := i.Send(msg); err != nil {
			// Forget this interface because it's not active anymore.
			deleted++
			s.Interfaces[index] = s.Interfaces[count-deleted]
		}
	}
	if deleted > 0 {
		// Ensure that we don't keep garbage references around.
		for index := deleted; index > 0; index-- {
			s.Interfaces[count-index] = nil
		}
		// Shorten the slice.
		s.Interfaces = s.Interfaces[:count-deleted]
	}
}
