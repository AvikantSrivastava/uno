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
	clients           map[string]*websocket.Conn
	upgrader          websocket.Upgrader
	broadcast         chan string
	syncChannel       chan string
	gameStarted       bool
	locks             map[string]*sync.Mutex
	wg                *sync.WaitGroup
	mu                sync.RWMutex
	broadcastStarted  bool
	broadcastStopChan chan struct{}
}

func NewNetwork() *Network {
	return &Network{
		clients: make(map[string]*websocket.Conn),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Accepts requests from every source
			},
		},
		broadcast:         make(chan string),
		gameStarted:       false,
		locks:             make(map[string]*sync.Mutex),
		wg:                &sync.WaitGroup{},
		broadcastStarted:  false,
		broadcastStopChan: make(chan struct{}),
	}
}

func (n *Network) AddClient(player game.Player, conn *websocket.Conn) {
	n.mu.Lock()
	defer n.mu.Unlock()
	// Close existing connection if any (for reconnection)
	if oldConn, exists := n.clients[player.Name]; exists {
		oldConn.Close()
	}
	n.clients[player.Name] = conn
	n.locks[player.Name] = &sync.Mutex{}
}

func (n *Network) RemoveClient(player game.Player) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if conn, exists := n.clients[player.Name]; exists {
		conn.Close()
	}
	delete(n.clients, player.Name)
	delete(n.locks, player.Name)
}

func (n *Network) GetClient(player game.Player) (*websocket.Conn, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	conn, ok := n.clients[player.Name]
	return conn, ok
}

func (n *Network) GetAllClients() map[string]*websocket.Conn {
	n.mu.RLock()
	defer n.mu.RUnlock()
	// Return a copy to avoid race conditions
	clientsCopy := make(map[string]*websocket.Conn, len(n.clients))
	for k, v := range n.clients {
		clientsCopy[k] = v
	}
	return clientsCopy
}

func (n *Network) StartBroadcaster() {
	n.mu.Lock()
	if n.broadcastStarted {
		n.mu.Unlock()
		return
	}
	n.broadcastStarted = true
	n.mu.Unlock()
	go n.BroadcastMessages()
}

func (n *Network) StopBroadcaster() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if !n.broadcastStarted {
		return
	}
	close(n.broadcastStopChan)
	close(n.broadcast)
	n.broadcastStarted = false
}

func (n *Network) BroadcastMessages() {
	for message := range n.broadcast {
		clients := n.GetAllClients()
		clientCount := len(clients)

		if clientCount == 0 {
			continue
		}

		// Increment the WaitGroup counter for the number of clients
		n.wg.Add(clientCount)

		// Broadcast the message to all players
		for playerName, conn := range clients {
			go func(playerName string, conn *websocket.Conn, message string) {
				defer n.wg.Done()

				n.mu.RLock()
				lock, exists := n.locks[playerName]
				n.mu.RUnlock()

				if !exists {
					return
				}

				lock.Lock()
				err := conn.WriteMessage(websocket.TextMessage, []byte(message))
				lock.Unlock()

				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						fmt.Printf("Unexpected close error for player %s: %v\n", playerName, err)
					} else {
						fmt.Printf("Error writing message to player %s: %v\n", playerName, err)
					}
					n.RemoveClientByName(playerName)
				}
			}(playerName, conn, message)
		}

		// Wait for all goroutines to finish broadcasting this message
		n.wg.Wait()
	}
}

// RemoveClientByName removes a client by player name
func (n *Network) RemoveClientByName(playerName string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if conn, exists := n.clients[playerName]; exists {
		conn.Close()
	}
	delete(n.clients, playerName)
	delete(n.locks, playerName)
}

func (n *Network) ListenToClient(player *game.Player, r *Room) {
	g := &r.game
	g.Network.StartBroadcaster()

	g.mu.Lock()
	clientCount := len(g.Network.clients)
	gameStarted := g.GameStarted
	g.mu.Unlock()

	if clientCount == r.maxPlayers && !gameStarted {
		g.Start()
		conn_info_dto := dtos.ConnectionDTO{
			PlayerName: player.Name,
			RoomID:     r.id,
			MaxPlayers: r.maxPlayers,
			Players:    r.game.getAllPlayers(),
		}
		go g.SyncAllPlayers()

		g.Network.SendMessage(player, conn_info_dto.Serialize())
		g.Network.BroadcastInfoMessage("Everyone's here! Let's play!")
	} else {
		g.Network.BroadcastInfoMessage("Waiting for other players to join...")
	}

	conn, ok := n.GetClient(*player)
	if !ok {
		return
	}

	defer func() {
		n.RemoveClient(*player)
		r.handlePlayerDisconnect(player)
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		g.HandleCommand(msg, player)
	}
}

func (n *Network) BroadcastMessage(message []byte) {
	n.broadcast <- string(message)
}

func (n *Network) SendMessage(p *game.Player, message []byte) error {
	n.mu.RLock()
	conn, connExists := n.clients[p.Name]
	lock, lockExists := n.locks[p.Name]
	n.mu.RUnlock()

	if !connExists {
		return fmt.Errorf("player %s not found in network clients", p.Name)
	}

	if !lockExists {
		return fmt.Errorf("no mutex found for player %s", p.Name)
	}

	lock.Lock()
	defer lock.Unlock()

	err := conn.WriteMessage(websocket.TextMessage, message)
	if err != nil {
		return fmt.Errorf("error sending message to player %s: %v", p.Name, err)
	}
	return nil
}

func (n *Network) SendInfoMessage(p *game.Player, message string) {
	if p == nil || !p.Connected {
		return // Don't send to disconnected players
	}
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
	n.mu.RLock()
	conn, exists := n.clients[p.Name]
	n.mu.RUnlock()
	if exists && conn != nil {
		conn.Close()
	}
}
