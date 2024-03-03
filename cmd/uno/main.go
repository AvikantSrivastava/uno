package main

import (
	"fmt"
	"net/http"
	"uno/internal"

	"github.com/gorilla/websocket"
)

var clients = make(map[*websocket.Conn]bool)
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true //Acceptes request from evry rsrc
	},
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()

	clients[conn] = true
	fmt.Println("New client joined")
	for {
		// Read message from browser
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}

		// Print the message to the console
		fmt.Printf("%s sent: %s\n", conn.RemoteAddr(), string(msg))

		for client, _ := range clients {
			// Write message back to browser
			if err = client.WriteMessage(msgType, msg); err != nil {
				return
			}
		}

	}

	// Handle game-related actions, e.g., sending initial game state to the player

	// Start a goroutine to handle messages from the player

}

func main() {
	players := []string{"lasan", "player2", "player3"}
	http.HandleFunc("/ws", handleConnections)
	game := internal.NewGame(players)

	//topCard := game.GameDeck.TopCard()
	//fmt.Println(topCard.LogCard())
	game.Start()

	//go handleMessages()

	fmt.Println("Server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic("Error starting server: " + err.Error())
	}
}
