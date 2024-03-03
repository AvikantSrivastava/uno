package internal

import (
	"fmt"
	"sync"

	"uno/models"
)

type Game struct {
	Players      []*models.Player
	GameDeck     *models.GameDeck
	CurrentTurn  int
	Reverse      bool
	ActivePlayer *models.Player //pointer to active player
	mu           sync.Mutex
	GameTopCard  *models.Card //TopCard of Playing Game
}

func NewGame(playerNames []string) *Game {
	players := make([]*models.Player, len(playerNames)) //Clients
	for i, name := range playerNames {
		players[i] = models.NewPlayer(name)
	}
	gameDeck := models.NewGameDeck() //Initialised Game Deck
	for _, p := range players {
		p.AddCards(gameDeck.Cut(7)) //Players get the cards
	}

	game := &Game{
		Players:  players,
		GameDeck: gameDeck,
	}
	return game
}
func (g *Game) Start() {
	// g.mu.Lock()
	// defer g.mu.Unlock()

	// Start the first player's turn
	g.CurrentTurn = 0
	g.ActivePlayer = g.Players[g.CurrentTurn] //g.Players is already a pointer
	g.ActivePlayer.CardInHand()
	//Commenting start
	//g.nextTurn()

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
	if g.IsValidMove(card) {
		// Perform game logic for playing the card
		// Update game state, check for UNO, etc.

		// Notify all players about the played card
		for _, p := range g.Players {
			p.Send(fmt.Sprintf("%s played %s", player.Name, card.LogCard()))
		}

		// Check if the player has won\
		//Set Game top card
		g.GameTopCard = &card //  assignment to a pointer
		// Move to the next turn
		g.nextTurn()
	} else {
		// Notify the player that the move is invalid
		player.Send("Invalid move. Try again.")
	}
}
func (g *Game) IsValidMove(playedcard models.Card) bool {

	GameTopCard := g.GameTopCard
	if playedcard.IsSameColor(*GameTopCard) || playedcard.IsSameRank(*GameTopCard) {
		return true
	}

	return false
}
