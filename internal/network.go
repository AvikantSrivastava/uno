package internal

import (
	"fmt"
	"net/http"
	"sync"
	"uno/models/dtos"
	"uno/models/game"

	"github.com/gorilla/websocket"
)

type Network struct {
	//clients map[*websocket.Conn]*models.Player
	clients     map[game.Player]*websocket.Conn
	upgrader    websocket.Upgrader
	broadcast   chan string
	syncChannel chan string
	gameStarted bool
	locks       map[game.Player]*sync.Mutex
	wg          *sync.WaitGroup
	mu          sync.RWMutex
}

func NewNetwork() *Network {
	return &Network{
		clients: make(map[game.Player]*websocket.Conn),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Accepts requests from every source
			},
		},
		broadcast:   make(chan string),
		gameStarted: false,
		locks:       make(map[game.Player]*sync.Mutex),
		wg:          &sync.WaitGroup{},
	}
}

func (n *Network) AddClient(player game.Player, conn *websocket.Conn) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.clients[player] = conn
	n.locks[player] = &sync.Mutex{}
}

func (n *Network) RemoveClient(player game.Player) {
	n.mu.Lock()
	defer n.mu.Unlock()
	delete(n.clients, player)
	log.Printf("%s Player is removed from map", player.Name)
}

func (n *Network) GetClient(player game.Player) (*websocket.Conn, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	conn, ok := n.clients[player]
	return conn, ok
}

func (n *Network) GetAllClients() map[game.Player]*websocket.Conn {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.clients
}

func (n *Network) BroadcastMessages() {
	for message := range n.broadcast {
		clientCount := len(n.clients)
		clients := n.GetAllClients()

		// Increment the WaitGroup counter for the number of clients
		if clientCount > 0 {
			n.wg.Add(clientCount)
		}

		// Broadcast the message to all players
		for player, conn := range clients {
			go func(player game.Player, conn *websocket.Conn, message string) {
				defer n.wg.Done()

				lock := n.locks[player]
				lock.Lock()
				defer lock.Unlock()

				err := conn.WriteMessage(websocket.TextMessage, []byte(message))
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						fmt.Printf("Unexpected close error for player %s: %v\n", player.Name, err)
					} else {
						fmt.Printf("Error writing message to player %s: %v\n", player.Name, err)
					}

					// Close the connection and remove the player from the map
					conn.Close()

					n.mu.Lock()
					defer n.mu.Unlock()
					delete(n.clients, player)
					delete(n.locks, player)
				}
			}(player, conn, message)
		}

		// Wait for all goroutines to finish broadcasting this message
		n.wg.Wait()
	}
}

func (n *Network) ListenToClient(player *game.Player, r *Room) {
	game := &r.game
	go game.Network.BroadcastMessages()

	if (len(game.Network.clients) == r.maxPlayers) && (game.GameStarted == false) {
		game.Start()
		conn_info_dto := dtos.ConnectionDTO{
			PlayerName: player.Name,
			RoomID:     r.id,
			MaxPlayers: r.maxPlayers,
			Players:    r.game.getAllPlayers(),
		}
		go game.SyncAllPlayers()

		game.Network.SendMessage(player, conn_info_dto.Serialize())
		game.Network.BroadcastInfoMessage("All players have joined. Game has started.")
	} else {
		game.Network.BroadcastInfoMessage("Waiting for players to join the game.")
	}
	conn := n.clients[*player]
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}

		go game.HandleCommand(msg, player)

	}
}

func (n *Network) BroadcastMessage(message []byte) {
	n.broadcast <- string(message)
}

func (n *Network) SendMessage(p *game.Player, message []byte) error {
	conn, exists := n.clients[*p]
	if !exists {
		return fmt.Errorf("player %s not found in network clients", p.Name)
	}

	lock, exists := n.locks[*p]
	if !exists {
		return fmt.Errorf("no mutex found for player %s", p.Name)
	}

	// Lock the mutex for the player's connection
	lock.Lock()
	defer lock.Unlock()

	// Perform the write operation
	err := conn.WriteMessage(websocket.TextMessage, message)
	if err != nil {
		return fmt.Errorf("error sending message to player %s: %v", p.Name, err)
	}
	return nil
}

func (n *Network) SendInfoMessage(p *game.Player, message string) {
	dto := dtos.InfoDTO{Message: message}
	n.SendMessage(p, dto.Serialize())
}

func (n *Network) BroadcastInfoMessage(message string) {
	dto := dtos.InfoDTO{Message: message}
	n.broadcast <- string(dto.Serialize())
}

func (n *Network) BroadcastConnectionInfo() {

}

func (n *Network) CloseConnection(p *game.Player) {
	conn := n.clients[*p]
	conn.Close()
}
