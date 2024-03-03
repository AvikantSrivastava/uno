package models

import (
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
			gd.AddCard(Card{Color: c, Rank: r})
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

func (gd *GameDeck) TopCard() *Card {
	return &gd.Cards[0]
}
