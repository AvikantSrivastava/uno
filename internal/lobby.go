package internal

import (
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"uno/models/dtos"
	"uno/models/game"

	"github.com/gorilla/websocket"
)

const (
	MaxPlayerNameLength = 32
	MinPlayerNameLength = 1
	MinPlayers          = 2
	MaxPlayersPerRoom   = 10
)

var playerNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// Room represents a game room
type Room struct {
	id         int
	game       Game
	maxPlayers int
	mu         sync.RWMutex
}

const (
	ROOM_START_INDEX = 1000
	ROOM_END_INDEX   = 9999
	MAX_ROOMS        = ROOM_END_INDEX - ROOM_START_INDEX
)

var (
	rooms   = make(map[int]*Room)
	roomsMu sync.RWMutex
)

func NewRoom(maxPlayers int) (*Room, error) {
	roomsMu.Lock()
	defer roomsMu.Unlock()

	if len(rooms) >= MAX_ROOMS {
		return nil, fmt.Errorf("Server is busy! Try again later")
	}

	roomId := generateUniqueID()
	if roomId == -1 {
		return nil, fmt.Errorf("Couldn't create room, please try again")
	}

	r := &Room{
		id:         roomId,
		game:       *NewGame(),
		maxPlayers: maxPlayers,
	}
	r.game.Room = r
	rooms[roomId] = r
	return r, nil
}

func (r *Room) handlePlayerDisconnect(player *game.Player) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Mark player as disconnected instead of removing
	player.Connected = false

	// Broadcast disconnect message
	r.game.Network.BroadcastInfoMessage(fmt.Sprintf("%s disconnected", player.Name))

	// Check if all players are disconnected
	allDisconnected := true
	for _, p := range r.game.Players {
		if p.Connected {
			allDisconnected = false
			break
		}
	}

	// If all players disconnected, cleanup the room
	if allDisconnected {
		r.cleanup()
	} else {
		// If it was their turn, move to next connected player
		r.game.SkipToNextConnectedPlayer()
	}
}

// FindDisconnectedPlayer finds a disconnected player by name in the room
func (r *Room) FindDisconnectedPlayer(name string) *game.Player {
	upperName := strings.ToUpper(name)
	for _, p := range r.game.Players {
		if p.Name == upperName && !p.Connected {
			return p
		}
	}
	return nil
}

// FindPlayerByName finds any player by name in the room
func (r *Room) FindPlayerByName(name string) *game.Player {
	upperName := strings.ToUpper(name)
	for _, p := range r.game.Players {
		if p.Name == upperName {
			return p
		}
	}
	return nil
}

func (r *Room) cleanup() {
	r.game.Network.StopBroadcaster()
	roomsMu.Lock()
	delete(rooms, r.id)
	roomsMu.Unlock()
}

func GetRoom(roomId int) (*Room, bool) {
	roomsMu.RLock()
	defer roomsMu.RUnlock()
	room, ok := rooms[roomId]
	return room, ok
}

func validatePlayerName(name string) error {
	if len(name) < MinPlayerNameLength {
		return fmt.Errorf("player name too short (min %d characters)", MinPlayerNameLength)
	}
	if len(name) > MaxPlayerNameLength {
		return fmt.Errorf("player name too long (max %d characters)", MaxPlayerNameLength)
	}
	if !playerNameRegex.MatchString(name) {
		return fmt.Errorf("player name contains invalid characters (only alphanumeric, underscore, hyphen allowed)")
	}
	return nil
}

// CreateRoomHandler handles requests to create a new room
func CreateRoomHandler(w http.ResponseWriter, r *http.Request) {
	playerName := r.URL.Query().Get("player_name")
	if err := validatePlayerName(playerName); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	maxPlayersStr := r.URL.Query().Get("max_players")
	if maxPlayersStr == "" {
		http.Error(w, "Missing max_players parameter", http.StatusBadRequest)
		return
	}

	maxPlayers, err := strconv.Atoi(maxPlayersStr)
	if err != nil {
		http.Error(w, "Invalid max_players parameter", http.StatusBadRequest)
		return
	}

	if maxPlayers < MinPlayers || maxPlayers > MaxPlayersPerRoom {
		http.Error(w, fmt.Sprintf("max_players must be between %d and %d", MinPlayers, MaxPlayersPerRoom), http.StatusBadRequest)
		return
	}

	room, err := NewRoom(maxPlayers)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	g := &room.game
	player, err := AddPlayerToRoom(room.id, playerName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	conn := UpgradeWebsocket(w, r, room)
	if conn == nil {
		return
	}

	g.Network.AddClient(*player, conn)

	dto := dtos.ConnectionDTO{
		PlayerName:  upperName,
		RoomID:      room.id,
		MaxPlayers:  maxPlayers,
		Players:     room.game.getAllPlayers(),
		IsReconnect: false,
	}
	if err := conn.WriteMessage(websocket.TextMessage, dto.Serialize()); err != nil {
		g.Network.RemoveClient(*player)
		return
	}

	g.Network.ListenToClient(player, room)
}

func JoinRoomHandler(w http.ResponseWriter, r *http.Request) {
	playerName := r.URL.Query().Get("player_name")
	if err := validatePlayerName(playerName); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	roomIdStr := r.URL.Query().Get("room_id")
	if roomIdStr == "" {
		http.Error(w, "room_id is required", http.StatusBadRequest)
		return
	}

	roomId, err := strconv.Atoi(roomIdStr)
	if err != nil {
		http.Error(w, "room_id must be a valid integer", http.StatusBadRequest)
		return
	}

	room, ok := GetRoom(roomId)
	if !ok {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}

	// Check if this is a reconnection attempt
	room.mu.Lock()
	existingPlayer := room.FindPlayerByName(playerName)

	var player *game.Player
	var isReconnect bool

	if existingPlayer != nil {
		if existingPlayer.Connected {
			room.mu.Unlock()
			http.Error(w, "This name is already taken in this room", http.StatusBadRequest)
			return
		}
		// Reconnection - reuse existing player
		player = existingPlayer
		player.Connected = true
		isReconnect = true
	} else {
		room.mu.Unlock()
		// New player joining
		var err error
		player, err = AddPlayerToRoom(roomId, playerName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		room.mu.Lock()
	}

	conn := UpgradeWebsocket(w, r, room)
	if conn == nil {
		return
	}

	g := &room.game
	g.Network.AddClient(*player, conn)

	dto := dtos.ConnectionDTO{
		PlayerName:  upperName,
		RoomID:      room.id,
		MaxPlayers:  room.maxPlayers,
		Players:     room.game.getAllPlayers(),
		IsReconnect: isReconnect,
	}
	if err := conn.WriteMessage(websocket.TextMessage, dto.Serialize()); err != nil {
		g.Network.RemoveClient(*player)
		return
	}

	if isReconnect {
		g.Network.BroadcastInfoMessage(fmt.Sprintf("%s is back!", upperName))
		// Sync the reconnected player with current game state
		g.SyncPlayer(player)
	}

	g.Network.ListenToClient(player, room)
}

func AddPlayerToRoom(roomId int, playerName string) (*game.Player, error) {
	room, ok := GetRoom(roomId)
	if !ok {
		return nil, fmt.Errorf("room not found")
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	g := &room.game
	if len(g.Players) >= room.maxPlayers {
		return nil, fmt.Errorf("room is full")
	}

	player := game.NewPlayer(playerName)
	g.AddPlayer(player)
	return player, nil
}

func UpgradeWebsocket(w http.ResponseWriter, r *http.Request, room *Room) *websocket.Conn {
	conn, err := room.game.Network.upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("Error upgrading to WebSocket: %v\n", err)
		return nil
	}
	return conn
}

// generateUniqueID generates a unique room id, avoiding collisions
// Must be called with roomsMu held
func generateUniqueID() int {
	if len(rooms) >= MAX_ROOMS {
		return -1
	}

	// Try random IDs first (faster in most cases)
	for i := 0; i < 100; i++ {
		id := ROOM_START_INDEX + rand.Intn(ROOM_END_INDEX-ROOM_START_INDEX+1)
		if _, exists := rooms[id]; !exists {
			return id
		}
	}

	// Fallback: linear search for available ID
	for id := ROOM_START_INDEX; id <= ROOM_END_INDEX; id++ {
		if _, exists := rooms[id]; !exists {
			return id
		}
	}

	return -1
}
