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

//Messagee ...
type Messagee struct {
	Type    string                 `json:"type"`
	Content map[string]interface{} `json:"content"`
}

var broadcast = make(chan Messagee)
var clients = make(map[*websocket.Conn]bool)

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
	clients[ws] = true
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

		broadcast <- sendData
		fmt.Printf("%+v\n", broadcast)
	}
}

func handleMessages() {
	for {
		//grab mess from broadcast
		sendData := <-broadcast
		fmt.Printf("%+v\n", sendData)
		//send data to every client
		for client := range clients {
			err := client.WriteJSON(sendData)
			if err != nil {
				log.Println(err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}
func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, " Home Page")
}

func setupRoutes() {
	http.HandleFunc("/", homePage)
	http.HandleFunc("/ws", wsEndpoint)
}

func main() {
	fmt.Println("Hello World")
	setupRoutes()
	go handleMessages()
	log.Fatal(http.ListenAndServe(":8080", nil))
}
