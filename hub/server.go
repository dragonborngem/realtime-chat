package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Hub struct {
	BroadCastTo chan Messagee
	Clients     map[*websocket.Conn]bool
}

func newHub() *Hub {
	return &Hub{
		BroadCastTo: make(chan Messagee),
		Clients:     make(map[*websocket.Conn]bool),
	}
}

//Messagee ...
type Messagee struct {
	Type    string                 `json:"type"`
	Content map[string]interface{} `json:"content"`
}

var hub = newHub()

func wsEndpoint(w http.ResponseWriter, r *http.Request) {
	// check origin, always = true for this demo
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	//upgrade connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		fmt.Println("a")
	}
	log.Println("client connected")
	err = ws.WriteMessage(1, []byte("Hi Client!"))
	if err != nil {
		log.Println(err)
		fmt.Println("b")
	}
	hub.Clients[ws] = true
	defer ws.Close()
	reader(ws)
}

func reader(conn *websocket.Conn) {
	for {
		//read in a message
		dataReceive := make(map[string]interface{})
		err := websocket.ReadJSON(conn, &dataReceive)
		if err != nil {
			return
		}

		//fmt.Printf("%+v\n", dataReceive)
		var sendData Messagee
		sendData.Type = "message"
		sendData.Content = make(map[string]interface{})
		sendData.Content["message"] = dataReceive["message"]
		sendData.Content["sender_name"] = dataReceive["sender_name"]

		hub.BroadCastTo <- sendData
		fmt.Printf("%+v\n", hub.BroadCastTo)
	}
}

func handleMessages() {
	for {
		//grab mess from broadcast
		sendData := <-hub.BroadCastTo
		fmt.Printf("%+v\n", sendData)
		//send data to every client
		for client := range hub.Clients {
			err := client.WriteJSON(sendData)
			if err != nil {
				log.Println(err)
				client.Close()
				delete(hub.Clients, client)
			}
		}
	}
}

// func homePage(w http.ResponseWriter, r *http.Request) {
// 	fmt.Fprintf(w, " Home Page")
// }

func setupRoutes() {
	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/ws", wsEndpoint)
}

func main() {
	fmt.Println("Hello World")
	setupRoutes()
	go handleMessages()
	log.Fatal(http.ListenAndServe(":8080", nil))
}
