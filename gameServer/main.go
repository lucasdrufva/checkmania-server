package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/lucasdrufva/checkmania-server/gameServer/messageQueue"
	"github.com/lucasdrufva/checkmania-server/gameServer/utilities"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Home Page")
}

var ms = &messageQueue.MessageQueue{}

func wsEndpoint(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	log.Println("client connected")

	client := messageQueue.SocketClient{
		Id:         utilities.AutoId(),
		Auth:       false,
		Connection: ws,
	}

	ms.AddClient(client)

	go ms.DeQueue()
	for {
		messageType, p, err := ws.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}

		ms.HandleRecieveMessage(client.Id, messageType, p)
	}
}

func setupRoutes() {
	http.HandleFunc("/", homePage)
	http.HandleFunc("/ws", wsEndpoint)
}

func main() {
	fmt.Println("hello world")
	setupRoutes()
	ms.Connect()
	log.Fatal(http.ListenAndServe(":8080", nil))
}
