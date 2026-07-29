package game

import (
	"errors"
	color "uno/models/constants/color"
	rank "uno/models/constants/rank"
)

type GameDeck struct {
	*Deck
}

func NewGameDeck() *GameDeck {
	gd := &GameDeck{
		Deck: NewDeck(),
	}
	gd.initColoredCards()
	gd.initNonColoredCards()
	gd.Shuffle()
	return gd
}

func (gd *GameDeck) initColoredCards() {
	for _, c := range color.ALLColors {
		for _, r := range rank.NumberCards {
			if r == "0" {
				gd.AddCard(Card{Color: c, Rank: r})
			} else {
				gd.AddCard(Card{Color: c, Rank: r})
				gd.AddCard(Card{Color: c, Rank: r})
			}

		}

		for _, r := range rank.ActionCards {
			gd.AddCard(Card{Color: c, Rank: r})
			gd.AddCard(Card{Color: c, Rank: r})
		}
	}
}

func (gd *GameDeck) initNonColoredCards() {
	for range [4]struct{}{} {
		for _, r := range rank.ActionCardsNoColor {
			gd.AddCard(Card{Color: "", Rank: r})
		}
	}
}

func (gd *GameDeck) Cut(n int) []Card {
	if n <= 0 || n > len(gd.Cards) {
		// If n is invalid, return an empty slice
		return []Card{}
	}
	cutCards := gd.Cards[:n]
	gd.Cards = gd.Cards[n:] // Remove the cut cards from the deck
	return cutCards
}

func (gd *GameDeck) GetStartCard() *Card {
	for i := range gd.Cards {
		if gd.Cards[i].Type() == "number-card" {
			card := gd.Cards[i]
			gd.Cards = append(gd.Cards[:i], gd.Cards[i+1:]...)
			return &card
		}
	}
	return nil
}

func (gd *GameDeck) TopCard() (*Card, error) {
	if len(gd.Cards) == 0 {
		return nil, errors.New("game deck is empty")
	}
	return &gd.Cards[0], nil
}
