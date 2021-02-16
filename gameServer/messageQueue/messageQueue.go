package messageQueue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"github.com/lucasdrufva/checkmania-server/gameServer/utilities"
)

var (
	PUBLISH      = "publish"
	SUBSCRIBE    = "subscribe"
	AUTHENTICATE = "auth"
	CREATE_GAME  = "createGame"
)

var ctx = context.Background()

type MessageQueue struct {
	Clients []SocketClient
	Games   []Game
	rdb     *redis.Client
}

type SocketClient struct {
	Id         string
	Auth       bool
	PlayerId   string
	Connection *websocket.Conn
}

type Message struct {
	Message  json.RawMessage `json:"message"`
	Action   string          `json:"action"`
	Token    string          `json:"token"`
	SenderId string
}

type Game struct {
	Id      string
	Started bool
	Players []string
}

func (ms *MessageQueue) Connect() *MessageQueue {
	ms.rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	err := ms.rdb.Set(ctx, "connected", "true", 0).Err()
	if err != nil {
		panic(err)
	}

	return ms
}

func (ms *MessageQueue) AddClient(client SocketClient) *MessageQueue {
	ms.Clients = append(ms.Clients, client)

	fmt.Println("adding new client to list", client.Id, len(ms.Clients))

	payload := []byte("Hello client: " + client.Id)
	client.Connection.WriteMessage(1, payload)
	return ms
}

func (ms *MessageQueue) UpdateClient(id string, client SocketClient) *MessageQueue {
	for i, v := range ms.Clients {
		if v.Id == id {
			ms.Clients[i] = client
		}
	}
	return ms
}

func (ms *MessageQueue) GetClient(id string) *SocketClient {
	//TODO: add error handling
	for i, v := range ms.Clients {
		if v.Id == id {
			return &ms.Clients[i]
		}
	}
	return nil
}

func (ms *MessageQueue) QueueMessage(m Message) *MessageQueue {
	toQueue, err := json.Marshal(m)
	if err != nil {
		fmt.Println("err boho")
		panic(err)
	}

	err = ms.rdb.LPush(ctx, "game", string(toQueue)).Err()
	if err != nil {
		fmt.Println("error here")
		panic(err)
	}
	return ms
}

func (ms *MessageQueue) Authenticate(client SocketClient, tokenString string) *MessageQueue {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		//Make sure that the token method conform to "SigningMethodHMAC"
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte("supersecret"), nil
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(token)

	claims := token.Claims.(jwt.MapClaims)
	fmt.Println(claims["id"].(string))

	client.Auth = true
	client.PlayerId = claims["id"].(string)

	ms.UpdateClient(client.Id, client)

	return ms
}

func (ms *MessageQueue) HandleRecieveMessage(clientId string, messageType int, payload []byte) *MessageQueue {
	client := *ms.GetClient(clientId)
	m := Message{}
	json.Unmarshal([]byte(payload), &m)
	m.SenderId = client.PlayerId

	fmt.Println("Client payload: ", m)
	//fmt.Println("raw payload: ", payload)

	switch m.Action {
	case PUBLISH:
		if client.Auth {
			ms.QueueMessage(m)
			fmt.Println("Pubslish new message")
		} else {
			fmt.Println("Unauthorised publish")
		}

	case AUTHENTICATE:
		fmt.Println("clients connected ", ms.Clients)
		ms.Authenticate(client, m.Token)
		fmt.Println("Authenticate client ", client.Id)
		fmt.Println("clients connected ", ms.Clients)

	case CREATE_GAME:
		fmt.Println("create game")
		ms.CreateGame(client)

	default:
	}

	return ms
}

func (ms *MessageQueue) DeQueue() *MessageQueue {
	for {
		time.Sleep(time.Second * 5)
		values, err := ms.rdb.LRange(ctx, "game", -1, -1).Result()
		if err != nil {
			fmt.Println("redis err")
			continue
		}
		if len(values) == 0 {
			fmt.Println("empty")
			continue
		}

		fmt.Println("last", values[0])

		res := Message{}
		json.Unmarshal([]byte(values[0]), &res)

		for _, client := range ms.Clients {
			if client.PlayerId != res.SenderId {
				_, err := ms.rdb.RPop(ctx, "game").Result()
				if err != nil {
					panic(err)
				}
				fmt.Println(client.PlayerId, res.Message)
				client.Connection.WriteMessage(1, []byte(values[0]))
			}
		}
	}
}

func (ms *MessageQueue) CreateGame(client SocketClient) *MessageQueue {
	game := Game{
		Id:      utilities.GenerateGameId(),
		Started: false,
		Players: []string{client.PlayerId},
	}
	ms.Games = append(ms.Games, game)

	returnMessage := map[string]string{"action": "joinedGame", "gameId": game.Id}
	jsonMessage, _ := json.Marshal(returnMessage)
	client.Connection.WriteMessage(1, []byte(jsonMessage))

	return ms
}
