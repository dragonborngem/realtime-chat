package main

import "fmt"

type message struct {
	data map[string]interface{}
	room string
}

type subscription struct {
	conn *connection
	room string
}

// Hub maintains the set of active connections and broadcasts messages to the
// connections.
type Hub struct {
	// Registered connections.
	rooms map[string]map[*connection]bool

	// Inbound messages from the connections.
	broadcast chan message

	// Register requests from the connections.
	register chan subscription

	// Unregister requests from connections.
	unregister chan subscription

	//
	connectRoom chan subscription
}

var h = Hub{
	broadcast:   make(chan message),
	register:    make(chan subscription),
	unregister:  make(chan subscription),
	connectRoom: make(chan subscription),
	rooms:       make(map[string]map[*connection]bool),
}

func (h *Hub) run() {
	for {
		select {
		case s := <-h.register:
			connections := h.rooms[s.room]
			if connections == nil {
				connections = make(map[*connection]bool)
				h.rooms[s.room] = connections
			}
			h.rooms[s.room][s.conn] = true
		case s := <-h.unregister:
			connections := h.rooms[s.room]
			if connections != nil {
				if _, ok := connections[s.conn]; ok {
					delete(connections, s.conn)
					close(s.conn.send)
					fmt.Println("close connection ", s.conn)
					if len(connections) == 0 {
						delete(h.rooms, s.room)
					}
				}
			}
		// case s := <-h.connectRoom:

		case m := <-h.broadcast:
			connections := h.rooms[m.room]
			for c := range connections {
				select {
				case c.send <- m.data:
				default:
					close(c.send)
					delete(connections, c)
					fmt.Println("cant send connection closed")
					if len(connections) == 0 {
						delete(h.rooms, m.room)
					}
				}
			}
		}
	}
}
