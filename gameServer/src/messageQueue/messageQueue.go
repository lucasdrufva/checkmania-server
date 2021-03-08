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
	JOIN_GAME    = "joinGame"
	MAKE_MOVE    = "makeMove"
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
	GameId   string          `json:"gameId"`
	SenderId string
}

type Game struct {
	Id      string
	Started bool
	Players []string
}

func (ms *MessageQueue) Connect() *MessageQueue {
	ms.rdb = redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
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

func (ms *MessageQueue) GetClientById(id string) *SocketClient {
	//TODO: add error handling
	for i, v := range ms.Clients {
		if v.Id == id {
			return &ms.Clients[i]
		}
	}
	return nil
}

func (ms *MessageQueue) GetClientByPlayerId(id string) *SocketClient {
	//TODO: add error handling
	for i, v := range ms.Clients {
		if v.PlayerId == id {
			return &ms.Clients[i]
		}
	}
	return nil
}

func (ms *MessageQueue) GetJoinedGames(PlayerId string) []string {
	games := []string{}

	for _, game := range ms.Games {
		for _, player := range game.Players {
			if player == PlayerId {
				games = append(games, game.Id)
			}
		}
	}

	return games
}

func (ms *MessageQueue) GetGame(id string) *Game {
	//TODO: add error handling
	for i, v := range ms.Games {
		if v.Id == id {
			return &ms.Games[i]
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
	client := *ms.GetClientById(clientId)
	m := Message{}
	json.Unmarshal([]byte(payload), &m)
	m.SenderId = client.PlayerId

	fmt.Println("Client payload: ", m)
	//fmt.Println("raw payload: ", payload)

	//TODO: refactor auth check
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

	case MAKE_MOVE:
		if client.Auth {
			ms.MakeMove(m)
			fmt.Println("make move")
		} else {
			fmt.Println("Unauthorised makeMove")
		}

	case JOIN_GAME:
		ms.JoinGame(m)

	default:
	}

	return ms
}

func (ms *MessageQueue) DeQueue() *MessageQueue {
	for {
		for _, game := range ms.Games {
			time.Sleep(time.Second * 5)
			values, err := ms.rdb.LRange(ctx, game.Id, -1, -1).Result()
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

			for _, player := range game.Players {
				if player != res.SenderId {
					_, err := ms.rdb.RPop(ctx, game.Id).Result()
					if err != nil {
						panic(err)
					}
					client := ms.GetClientByPlayerId(player)
					fmt.Println(client.PlayerId, res.Message)
					client.Connection.WriteMessage(1, []byte(values[0]))
				}
			}
		}
	}
}

func (ms *MessageQueue) CreateGame(client SocketClient) *MessageQueue {
	game := Game{
		Id:      utilities.GenerateGameId(),
		Started: false,
		Players: []string{
			client.PlayerId,
		},
	}
	ms.Games = append(ms.Games, game)

	ms.joinedGameResponse(client.PlayerId, game.Id)

	return ms
}

//TODO: suport short gameId
func (ms *MessageQueue) JoinGame(m Message) *MessageQueue {
	for i := range ms.Games {
		if ms.Games[i].Id == m.GameId {
			ms.Games[i].Players = append(ms.Games[i].Players, m.SenderId)
			ms.joinedGameResponse(m.SenderId, m.GameId)
			break
		}
	}

	return ms
}

func (ms *MessageQueue) joinedGameResponse(playerId string, gameId string) *MessageQueue {

	returnMessage := map[string]string{
		"action": "joinedGame",
		"gameId": gameId,
	}

	jsonMessage, _ := json.Marshal(returnMessage)

	client := ms.GetClientByPlayerId(playerId)
	client.Connection.WriteMessage(1, []byte(jsonMessage))

	return ms
}

func (ms *MessageQueue) MakeMove(m Message) *MessageQueue {
	toQueue, err := json.Marshal(m)
	if err != nil {
		fmt.Println("err boho")
		panic(err)
	}

	err = ms.rdb.LPush(ctx, m.GameId, string(toQueue)).Err()
	if err != nil {
		fmt.Println("error here")
		panic(err)
	}
	return ms
}
