package internal

import (
	"fmt"
	"math/rand"
	"time"
	"uno/models"

	"golang.org/x/exp/slices"
)

func ShouldBroadcast(msg string) bool {
	privateCommands := []string{"showcards,playcard"}

	for _, command := range privateCommands {
		if command == msg {
			return false
		}
	}

	return true
}
func findPlayer(players []*models.Player, name string) *models.Player {
	idx := slices.IndexFunc(players, func(p *models.Player) bool {
		return p.Name == name
	})
	if idx == -1 {
		return nil
	}
	return players[idx]
}
func convertDirectionToInteger(direction bool) int {
	if direction {
		return 1
	}
	return -1
}
func removeCardFromHand(hand []models.Card, card models.Card) []models.Card {
	for i, c := range hand {
		if c == card {
			return append(hand[:i], hand[i+1:]...)
		}
	}
	return hand
}

// performDraw2Action performs the Draw2 action
func (g *Game) performDraw2Action(player *models.Player) {
	nextPlayer := g.getNextPlayer()
	cardsDrawn := g.GameDeck.Cut(2)
	nextPlayer.AddCards(cardsDrawn)
	nextPlayer.Send(fmt.Sprintf("Drew 2 cards due to %s's Draw2 action", player.Name))
}
func (g *Game) ShuffleDiscardPileToDeck() {
	if len(g.DisposedGameDeck.Deck.Cards) > 0 {
		// Shuffle the discard pile
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(g.DisposedGameDeck.Deck.Cards), func(i, j int) {
			g.DisposedGameDeck.Deck.Cards[i], g.DisposedGameDeck.Deck.Cards[j] = g.DisposedGameDeck.Deck.Cards[j], g.DisposedGameDeck.Deck.Cards[i]
		})

		// Create a new deck from the shuffled discard pile
		g.GameDeck.Deck.Cards = g.DisposedGameDeck.Deck.Cards
		g.GameDeck.Deck.Counter = g.DisposedGameDeck.Deck.Counter

		// Clear the discard pile
		g.DisposedGameDeck.Deck.Cards = make([]models.Card, 0)
	}
}

// reverseGameDirection reverses the game direction
func (g *Game) reverseGameDirection() {
	g.GameDirection = !g.GameDirection
}

// skipNextTurn skips the next player's turn
func (g *Game) skipNextTurn() {
	g.NextTurn()
	g.ActivePlayer.Send("Your turn was skipped")
}

// declareWinner declares the winner of the game
func (g *Game) declareWinner(winner *models.Player) {
	for _, p := range g.Players {
		p.Send(fmt.Sprintf("%s has won the game!", winner.Name))
	}
	// Perform any necessary  end-game animation with bubbleTea
}

// getNextPlayer returns the next player based on the game direction
func (g *Game) getNextPlayer() *models.Player {
	integerDirection := convertDirectionToInteger(g.GameDirection)
	nextTurn := (g.CurrentTurn + integerDirection) % len(g.Players)
	return g.Players[nextTurn]
}
