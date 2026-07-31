package dtos

import (
	"uno/models/constants/color"
	"uno/models/game"
)

// CardStatus represents how many cards a player has (without revealing exact count)
type CardStatus string

const (
	CardStatusTooMany CardStatus = "too_many"
	CardStatusMany    CardStatus = "many"
	CardStatusFew     CardStatus = "few"
	CardStatusVeryFew CardStatus = "very_few"
	CardStatusOne     CardStatus = "one"
)

// GetCardStatus returns the card status based on card count
func GetCardStatus(cardCount int) CardStatus {
	switch {
	case cardCount >= 10:
		return CardStatusTooMany
	case cardCount >= 7:
		return CardStatusMany
	case cardCount >= 4:
		return CardStatusFew
	case cardCount >= 2:
		return CardStatusVeryFew
	default:
		return CardStatusOne
	}
}

// PlayerInfo contains player information visible to other players
type PlayerInfo struct {
	Name       string     `json:"name"`
	CardStatus CardStatus `json:"card_status"`
	IsUno      bool       `json:"is_uno"` // true when player has 1 card left
}

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
