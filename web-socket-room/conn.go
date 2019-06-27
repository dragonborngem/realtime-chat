package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// connection is an middleman between the websocket connection and the hub.
type connection struct {
	// The websocket connection.
	ws *websocket.Conn

	// Buffered channel of outbound messages.
	send chan map[string]interface{}
}

// readPump pumps messages from the websocket connection to the hub.
func (s subscription) readPump() {
	c := s.conn
	defer func() {
		h.unregister <- s
		c.ws.Close()
	}()
	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		fmt.Printf("%+v\n", c)
		var receiveMessage map[string]interface{}
		err := c.ws.ReadJSON(&receiveMessage)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("error: %v", err)
			}
			fmt.Println("err read new mess")
			fmt.Println(err)
			break
		}
		if receiveMessage["type"] == "connect_room" {
			newSub := subscription{
				conn: s.conn,
			}
			newSub.room = fmt.Sprintf("%v", receiveMessage["room"])
			//unregister in current room, dont close connection
			connections := h.rooms[s.room]
			if connections != nil {
				if _, ok := connections[s.conn]; ok {
					delete(h.rooms[s.room], s.conn)
					fmt.Println("leave current room and connect to new", s.conn)
					if len(connections) == 0 {
						delete(h.rooms, s.room)
					}
				}
			}
			//register in new room
			connections = h.rooms[newSub.room]
			if connections == nil {
				connections = make(map[*connection]bool)
				h.rooms[newSub.room] = connections
			}
			h.rooms[newSub.room][s.conn] = true
			s.room = newSub.room
			c = s.conn
			// change hereeeeeeeeeeeeeee
		} else {
			m := message{receiveMessage, s.room}
			h.broadcast <- m
		}
	}
}

// write writes a message with the given message type and payload.
func (c *connection) write(mt int, payload []byte) error {
	//c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(mt, payload)
}

// writePump pumps messages from the hub to the websocket connection.
func (s *subscription) writePump() {
	c := s.conn
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		fmt.Println("here")
		c.ws.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.write(websocket.CloseMessage, []byte{})
				fmt.Println("here 1")
				return
			}
			if err := c.ws.WriteJSON(message); err != nil {
				fmt.Println(err)
				fmt.Println("here 2")
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				fmt.Println("here 3")
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func serveWs(w http.ResponseWriter, r *http.Request) {
	//upgrade connection to websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	vars := mux.Vars(r)
	fmt.Println(vars["room"])
	if err != nil {
		log.Println(err)
		return
	}
	c := &connection{send: make(chan map[string]interface{}, 256), ws: ws}
	s := subscription{c, vars["room"]}
	h.register <- s
	go s.writePump()
	s.readPump()
}
