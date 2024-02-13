package internal

import (
	"fmt"
	"models"
	"sync"
)

type Game struct {
	Players      []*models.Player
	GameDeck     *models.GameDeck
	CurrentTurn  int
	Reverse      bool
	ActivePlayer models.Player
	mu           sync.Mutex
}

func NewGame(playerNames []string) *Game {
	players := make([]*models.Player, len(playerNames))
	for i, name := range playerNames {
		players[i] = models.NewPlayer(name)
	}
	gameDeck := models.NewGameDeck()
	for _, p := range players {
		p.AddCards(gameDeck.Cut(7))
	}

	game := &Game{
		Players:  players,
		GameDeck: gameDeck,
	}
	return game
}
func (g *Game) Start() {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Shuffle and deal cards to players
	g.GameDeck.Shuffle()
	for i := 0; i < 7; i++ {
		for _, player := range g.Players {
			player.AddCard(g.GameDeck.DrawCard())
		}
	}

	// Start the first player's turn
	g.nextTurn()
}

func (g *Game) nextTurn() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.ActivePlayer != nil {
		g.ActivePlayer.Send("Your turn is over.")
	}

	g.CurrentTurn = (g.CurrentTurn + 1) % len(g.Players)
	g.ActivePlayer = g.Players[g.CurrentTurn]

	g.ActivePlayer.Send(fmt.Sprintf("It's your turn, %s!", g.ActivePlayer.Name))
}
func (g *Game) PlayCard(player *models.Player, card models.Card) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Check if the card is playable
	//Might remove this isValidMove to another package
	if g.GameDeck.IsValidMove(card) {
		// Perform game logic for playing the card
		// Update game state, check for UNO, etc.

		// Notify all players about the played card
		for _, p := range g.Players {
			p.Send(fmt.Sprintf("%s played %s", player.Name, card.LogCard()))
		}

		// Check if the player has won

		// Move to the next turn
		g.nextTurn()
	} else {
		// Notify the player that the move is invalid
		player.Send("Invalid move. Try again.")
	}
}
