// server.go
package main

import (
	"fmt"
	"log"
	"net/http"

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

	fmt.Printf("socketHandler")

	// should not be always true
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	// Upgrade our raw HTTP connection to a websocket based one
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("Error during connection upgradation:", err)
		return
	}
	defer conn.Close()

	conns = append(conns, conn)

	// The event loop
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error during message reading:", err)
			break
		}

		fmt.Println("message type", messageType)

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
			_, err = fmt.Sscan(string(message), &action, &channelId, &userId, text)
			if err != nil {
				fmt.Println("Incorrect params of command", string(message))
				continue
			}
			messegeReceived <- roomMessage{roomId: channelId, userId: userId, text: text}

		}

		/*
			log.Printf("Received: %s", message)
			err = conn.WriteMessage(messageType, message)
			if err != nil {
				log.Println("Error during message writing:", err)
				break
			}
		*/
	}
}

/*
func home(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Index Page")
}
*/

var roomCreated chan int
var userJoined chan roomAction
var userLeft chan roomAction
var messegeReceived chan roomMessage

var conns []*websocket.Conn

func sendMessage(message string) {
	for _, c := range conns {
		err := c.WriteMessage(1, []byte(message))
		if err != nil {
			fmt.Println("error when sending message", err)
		}
	}
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

			_, ok := rooms[roomId]
			if ok {
				fmt.Println("Room already exists", roomId)
				continue
			}

			rooms[roomId] = NewRoom(roomId)
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

			_, ok := rooms[a.roomId]
			if !ok {
				fmt.Println("No such room", a.roomId)
				continue
			}

			room := rooms[a.roomId]
			room.users[a.userId] = true

			// each iteration probably should be placed in seperate goroutine/channel
			sendMessage(fmt.Sprintln("joined", a.roomId, a.userId))
		}
	}()

	http.HandleFunc("/socket", socketHandler)
	//	http.HandleFunc("/", home)
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}
