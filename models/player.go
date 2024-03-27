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
	//mu   sync.Mutex
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
	fmt.Sprintf("Success writing conn for %s", p.Name)
}
func (p *Player) Send(message string) error {
	// Check if the connection is nil
	// p.mu.Lock()
	// defer p.mu.Unlock()
	if p.conn == nil {
		return fmt.Errorf("connection is nil for player: %s", p.Name)
	}

	// Send the message through the WebSocket connection
	err := p.conn.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		return fmt.Errorf("error sending message to player %s: %v", p.Name, err)
	}
	return nil

}
