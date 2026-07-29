package internal

import (
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
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
		return nil, fmt.Errorf("maximum number of rooms reached")
	}

	roomId := generateUniqueID()
	if roomId == -1 {
		return nil, fmt.Errorf("failed to generate unique room ID")
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

	// Remove player from game
	r.game.RemovePlayer(player)

	// If no players left, cleanup the room
	if len(r.game.Players) == 0 {
		r.cleanup()
	}
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
		PlayerName: playerName,
		RoomID:     room.id,
		MaxPlayers: maxPlayers,
		Players:    room.game.getAllPlayers(),
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

	player, err := AddPlayerToRoom(roomId, playerName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	conn := UpgradeWebsocket(w, r, room)
	if conn == nil {
		return
	}

	g := &room.game
	g.Network.AddClient(*player, conn)

	dto := dtos.ConnectionDTO{
		PlayerName: playerName,
		RoomID:     room.id,
		MaxPlayers: room.maxPlayers,
		Players:    room.game.getAllPlayers(),
	}
	if err := conn.WriteMessage(websocket.TextMessage, dto.Serialize()); err != nil {
		g.Network.RemoveClient(*player)
		return
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
