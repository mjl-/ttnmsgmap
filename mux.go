package main

type Register struct {
	Message  chan interface{}
	Response chan int64
}

type Mux struct {
	Register   chan Register
	Unregister chan int64
	Input      chan interface{}
}

func NewMux() *Mux {
	mux := &Mux{
		Register:   make(chan Register),
		Unregister: make(chan int64),
		Input:      make(chan interface{}, 1),
	}

	go func() {
		clients := map[int64]chan interface{}{}
		lastId := int64(0)
		for {
			select {
			case r := <-mux.Register:
				lastId += 1
				clients[lastId] = r.Message
				r.Response <- lastId

			case id := <-mux.Unregister:
				delete(clients, id)

			case msg := <-mux.Input:
				for _, c := range clients {
					select {
					case c <- msg:
					default:
					}
				}
			}
		}
	}()

	return mux
}
