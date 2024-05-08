package models

import (
	"fmt"
	"strings"
)

type Player struct {
	Name string
	*Deck
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

func (p *Player) HasPlayableCard(topCard *Card) bool {
	if topCard == nil {
		// If topCard is a null pointer i.e first card is draw or wildcard, return true
		return true
	}

	for _, card := range p.Deck.Cards {
		switch {
		case card.Type() == "action-card" && card.Color == topCard.Color: // DRAW2 ,SKIP ,REVERSE don't have rank
			return true
		case card.Type() == "action-card-no-color":
			return true
		case card.Color == topCard.Color || card.Rank == topCard.Rank:
			return true
		}
	}

	return false
}
