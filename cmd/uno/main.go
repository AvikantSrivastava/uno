package main

import (
	"fmt"
	"net/http"
	"uno/internal"

	"github.com/gorilla/websocket"
)

var clients = make(map[*websocket.Conn]string)
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Accepts requests from every source
	},
}
var players = []string{"lasan", "player2", "player3"}
var broadcast = make(chan string)

func broadcastMessages() {
	for {
		select {
		case message := <-broadcast:
			for client, _ := range clients {
				// Write message back to all clients' browser/terminal
				if err := client.WriteMessage(websocket.TextMessage, []byte(": "+message)); err != nil {
					return
				}
			}
		}
	}
}
func handleConnections(w http.ResponseWriter, r *http.Request, game *internal.Game) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		delete(clients, conn)
		conn.Close()
	}()

	// Ask the user to enter their name
	var clientName string
	for {
		if err := conn.WriteMessage(websocket.TextMessage, []byte("Please enter your name: ")); err != nil {
			fmt.Println(err)
			return
		}
		_, name, err := conn.ReadMessage()
		if err != nil {
			fmt.Println(err)
			return
		}

		// Check if the name exists in the players slice
		nameFound := false
		for _, p := range players {
			if p == string(name) {
				nameFound = true
				break
			}
		}

		if nameFound {
			// Name found, assign with ClientName Map
			clientName = string(name)
			break
		} else {
			// Name not found, ask the user to input again
			if err := conn.WriteMessage(websocket.TextMessage, []byte("Name not found. Please try again.")); err != nil {
				fmt.Println(err)
				return
			}
		}
	}

	clients[conn] = clientName

	fmt.Printf("New client '%s' joined\n", clientName)

	go broadcastMessages()

	for {
		// Read message from browser/terminal
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}

		//  add your own logic here to filter out certain messages
		game.HandleMessage(string(msg), conn, clientName)

		// before broadcasting them to other clients
		if internal.ShouldBroadcast(string(msg)) {
			// Print the message to the console
			fmt.Printf("%s sent: %s\n", clientName, string(msg))
			broadcast <- fmt.Sprintf("%s: %s", clientName, string(msg))
		}
	}
}

func main() {
	game := internal.NewGame(players)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleConnections(w, r, game)
	}) //Game is passed arg because handlemessages will need to use game.players

	// topCard := game.GameDeck.TopCard()
	// fmt.Println(topCard.LogCard())
	game.Start()
	fmt.Println("Server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic("Error starting server: " + err.Error())
	}
}
