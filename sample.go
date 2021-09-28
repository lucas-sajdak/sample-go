// server.go
package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type room struct {
	id    int
	users map[int]bool
}

func NewRoom(id int) room {
	r := room{id: id}
	r.users = make(map[int]bool)
	return r
}

/*
type user struct {
	//	id int
}
*/

/*
type message struct {
	//	text string
}
*/

type roomAction struct {
	roomId int
	userId int
}

type roomMessage struct {
	roomId int
	userId int
	text   string
}

var upgrader = websocket.Upgrader{} // use default options

var rooms map[int]room

func socketHandler(w http.ResponseWriter, r *http.Request) {
	// should not be always true !!!!
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	// Upgrade our raw HTTP connection to a websocket based one
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("Error during connection upgradation:", err)
		return
	}
	defer conn.Close()

	{
		mu.Lock()
		conns = append(conns, conn)
		mu.Unlock()
	}

	// The event loop
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error during message reading:", err)
			break
		}

		var action string
		_, err = fmt.Sscan(string(message), &action)
		if err != nil {
			fmt.Println("Unknown command", message)
			continue
		}

		switch action {
		case "create":
			var channelId int
			_, err = fmt.Sscan(string(message), &action, &channelId)
			if err != nil {
				fmt.Println("Incorrect params of command", string(message))
				continue
			}
			roomCreated <- channelId

		case "join", "leave":
			var userId, channelId int
			_, err = fmt.Sscan(string(message), &action, &channelId, &userId)
			if err != nil {
				fmt.Println("Incorrect params of command", string(message))
				continue
			}

			if action == "join" {
				userJoined <- roomAction{roomId: channelId, userId: userId}
			} else {
				userLeft <- roomAction{roomId: channelId, userId: userId}
			}

		case "send":
			var userId, channelId int
			var text string
			_, err = fmt.Sscan(string(message), &action, &channelId, &userId, &text)
			if err != nil {
				fmt.Println("Incorrect params of command", string(message))
				continue
			}
			messegeReceived <- roomMessage{roomId: channelId, userId: userId, text: text}

		}
	}
}

var roomCreated chan int
var userJoined chan roomAction
var userLeft chan roomAction
var messegeReceived chan roomMessage

var conns []*websocket.Conn

var mu sync.Mutex
var roomMu sync.Mutex

func sendMessage(message string) {
	mu.Lock()
	for _, c := range conns {
		err := c.WriteMessage(1, []byte(message))
		if err != nil {
			fmt.Println("error when sending message", err)
		}
	}
	mu.Unlock()
}

func main() {

	roomCreated = make(chan int)
	userJoined = make(chan roomAction)
	userLeft = make(chan roomAction)
	messegeReceived = make(chan roomMessage)

	rooms = make(map[int]room)

	go func() {
		for {
			roomId, opened := <-roomCreated
			if !opened {
				break
			}

			{
				roomMu.Lock()
				_, ok := rooms[roomId]
				if ok {
					fmt.Println("Room already exists", roomId)
					roomMu.Unlock()
					continue
				}

				rooms[roomId] = NewRoom(roomId)
				roomMu.Unlock()
			}
			// each iteration probably should be placed in seperate goroutine
			sendMessage(fmt.Sprintln("created", roomId))
		}
	}()

	go func() {
		for {
			a, opened := <-userJoined
			if !opened {
				break
			}

			{
				roomMu.Lock()
				_, ok := rooms[a.roomId]
				if !ok {
					fmt.Println("No such room", a.roomId)
					roomMu.Unlock()
					continue
				}

				room := rooms[a.roomId]
				room.users[a.userId] = true
				roomMu.Unlock()
			}

			// each iteration probably should be placed in seperate goroutine/channel
			sendMessage(fmt.Sprintln("joined", a.roomId, a.userId))
		}
	}()

	go func() {
		for {
			a, opened := <-userLeft
			if !opened {
				break
			}

			{
				roomMu.Lock()
				_, ok := rooms[a.roomId]
				if !ok {
					fmt.Println("No such room", a.roomId)
					roomMu.Unlock()
					continue
				}

				room := rooms[a.roomId]
				_, ok = room.users[a.userId]
				{
					if !ok {
						fmt.Println("User not in the channel", a.roomId)
						roomMu.Unlock()
						continue
					}
				}

				delete(room.users, a.userId)
				roomMu.Unlock()
			}

			// each iteration probably should be placed in seperate goroutine/channel
			sendMessage(fmt.Sprintln("left", a.roomId, a.userId))
		}
	}()
	go func() {
		for {
			m, opened := <-messegeReceived
			if !opened {
				break
			}

			{
				roomMu.Lock()
				_, ok := rooms[m.roomId]
				if !ok {
					fmt.Println("No such room", m.roomId)
					roomMu.Unlock()
					continue
				}

				room := rooms[m.roomId]
				_, ok = room.users[m.userId]
				if !ok {
					fmt.Println("No such user in room", m.roomId)
					roomMu.Unlock()
					continue
				}
				roomMu.Unlock()
			}

			// each iteration probably should be placed in seperate goroutine/channel
			sendMessage(fmt.Sprintln("sent", m.roomId, m.userId, m.text))
		}
	}()

	http.HandleFunc("/socket", socketHandler)
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}
