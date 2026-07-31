package dtos

import (
	"uno/models/constants/color"
	"uno/models/game"
)

type SyncDTO struct {
	Player game.Player `json:"player"`
	Game   GameState   `json:"game"`
	Room   RoomState   `json:"room"`
}

type GameState struct {
	TopCard  game.Card   `json:"topcard"`
	TopColor color.Color `json:"topcolor"`
	Turn     string      `json:"turn"`
	Reverse  bool        `json:"reverse"`
}

type RoomState struct {
	Players    []PlayerInfo `json:"players"`
	RoomId     int          `json:"id"`
	MaxPlayers int          `json:"max_players"`
}

func (dto SyncDTO) Serialize() []byte {
	return Serialize(
		dto, "sync")
}
