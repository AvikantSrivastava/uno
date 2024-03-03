package models

import (
	"math/rand"
	"time"
)

type Deck struct {
	Cards   []Card
	Counter map[string]int
}

func NewDeck() *Deck {

	return &Deck{
		Cards:   make([]Card, 0),
		Counter: make(map[string]int),
	}
}

func (d *Deck) Shuffle() {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(d.Cards), func(i, j int) {
		d.Cards[i], d.Cards[j] = d.Cards[j], d.Cards[i]
	})
}

func (d *Deck) AddCard(card Card) {
	d.Cards = append(d.Cards, card)
	d.Counter[card.Type()]++
}

func (d *Deck) RemoveCard(index int) Card {
	if index < 0 || index >= len(d.Cards) {
		return Card{} // Return an empty card or handle error
	}
	card := d.Cards[index]
	d.Cards = append(d.Cards[:index], d.Cards[index+1:]...)
	d.Counter[card.Type()]--
	return card
}

func (d *Deck) NumberOfCards() int {
	return len(d.Cards)
}
