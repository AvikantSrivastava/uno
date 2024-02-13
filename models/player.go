package models

import (
	"fmt"

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

func (player *Player) CardInHand() {
	fmt.Println(player.Name, "has")
	for i, c := range player.Deck.Cards {
		fmt.Println("Card", i, ":\t", c.LogCard())
	}
}

func (player *Player) AddCards(cards []Card) {
	for _, c := range cards {
		player.AddCard(c)
	}
}
func (p *Player) Send(message string) error {
	return p.conn.WriteMessage(websocket.TextMessage, []byte(message))
}
