package game

import (
	"fmt"
	"strings"
)

type Player struct {
	Name string
	*Deck
	Drawn bool
}

func NewPlayer(name string) *Player {
	player := &Player{
		Name: name,
		Deck: NewDeck(),
		Drawn: false,
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

func (p *Player) HasPlayableCard(topCard Card) bool {
	for _, card := range p.Deck.Cards {
		switch {
		case card.Type() == "action-card" && card.Color == topCard.Color: // DRAW2 ,SKIP ,REVERSE don't have rank
			return true
		case topCard.Type() == "action-card-no-color": //When topcard is of action-card-no-color type, return true only if
			if card.Color == TopColor || card.Type() == "action-card-no-color" { // card deck has matching color or deck has another no-color action card
				return true
			}
		case card.Color == topCard.Color || card.Rank == topCard.Rank:
			return true
		}
	}

	return false
}
