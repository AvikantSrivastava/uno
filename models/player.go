package models

import (
	"fmt"
	"strings"

	"github.com/gorilla/websocket"
)

type Player struct {
	Name string
	*Deck
	conn *websocket.Conn
}

func NewPlayer(name string) *Player {
	player := &Player{
		Name: name,
		Deck: NewDeck(),
	}
	return player
}

func (player *Player) CardInHand() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s has:\n", player.Name))
	for i, c := range player.Deck.Cards {
		sb.WriteString(fmt.Sprintf("Card %d:\t%s\n", i, c.LogCard()))
	}
	return sb.String()
}

func (player *Player) AddCards(cards []Card) {
	for _, c := range cards {
		player.AddCard(c)
	}
}
func (p *Player) SetConn(conn *websocket.Conn) {
	p.conn = conn
	fmt.Println("Success writing conn")
}
func (p *Player) Send(message string) error {
	return p.conn.WriteMessage(websocket.TextMessage, []byte(message))
}
