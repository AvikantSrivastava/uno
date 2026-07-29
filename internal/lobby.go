package internal

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"
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
	if len(rooms) >= MAX_ROOMS {
		http.Error(w, "Maximum number of rooms reached", http.StatusForbidden)
		return
	}

	// Get query parameters
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
	room := NewRoom(maxPlayers)

	game := &room.game
	player := AddPlayerToRoom(&w, room.id, playerName)

	// Respond with the room id
	conn := UpgradeWebsocket(w, r, room)
	game.Network.AddClient(*player, conn)

	dto := dtos.ConnectionDTO{
		PlayerName: playerName,
		RoomID: room.id,
		MaxPlayers: maxPlayers,
		Players: room.game.getAllPlayers(),
	}
	conn.WriteMessage(websocket.TextMessage, dto.Serialize())

	game.Network.ListenToClient(player, room)

}

func JoinRoomHandler(w http.ResponseWriter, r *http.Request) {
	playerName := r.URL.Query().Get("player_name")
	roomIdStr := r.URL.Query().Get("room_id")

	// Check if player_name and room_id are provided
	if playerName == "" || roomIdStr == "" {
		http.Error(w, "player_name and room_id are required", http.StatusBadRequest)
		return
	}

	// Validate room id
	roomId, err := strconv.Atoi(roomIdStr)
	if err != nil {
		http.Error(w, "room_id must be a valid integer", http.StatusBadRequest)
		return
	}
	room := rooms[roomId]
	game := &room.game
	player := AddPlayerToRoom(&w, roomId, playerName)
	conn := UpgradeWebsocket(w, r, room)
	game.Network.AddClient(*player, conn)

	dto := dtos.ConnectionDTO{
		PlayerName: playerName,
		RoomID: room.id,
		MaxPlayers: room.maxPlayers,
		Players: room.game.getAllPlayers(),
	}
	conn.WriteMessage(websocket.TextMessage, dto.Serialize())

	game.Network.ListenToClient(player, room)
}

func AddPlayerToRoom(w *http.ResponseWriter, roomId int, playerName string) *game.Player {
	r, ok := rooms[roomId]
	if !ok {
		http.Error(*w, "Room not found", http.StatusNotFound)
		return nil
	}
	g := &r.game
	if len(g.Players) >= r.maxPlayers {
		http.Error(*w, "Room is full", http.StatusForbidden)
		return nil
	}

	player := game.NewPlayer(playerName)
	g.AddPlayer(player)
	return player
}

func UpgradeWebsocket(w http.ResponseWriter, r *http.Request, room *Room) *websocket.Conn {
	conn, err := room.game.Network.upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading to WebSocket:", err)
		return nil
	}
	return conn
}

// Generate a unique room id
func generateID() int {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return ROOM_START_INDEX + r.Intn(ROOM_END_INDEX-ROOM_START_INDEX+1)
}
