package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/lucasdrufva/checkmania-server/gameServer/messageQueue"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func autoId() string {
	return uuid.Must(uuid.NewRandom()).String()
}

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Home Page")
}

func reader(conn *websocket.Conn) {
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}

		log.Println(string(p))
		log.Println(autoId())

		if err := conn.WriteMessage(messageType, p); err != nil {
			log.Println(err)
			return
		}
	}
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
		Id:         autoId(),
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
