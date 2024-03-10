package internal

import (
	"fmt"
	"strings"
	"sync"
	"uno/models"

	"github.com/gorilla/websocket"
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

func (g *Game) NextTurn() {
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
		g.NextTurn()
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
func (g *Game) HandleMessage(msg string, conn *websocket.Conn, clientName string) {
	parts := strings.Split(msg, " ")
	command := parts[0]

	// Find the player index in the Players slice
	playerPtr := findPlayer(g.Players, clientName)
	if playerPtr == nil {
		// Handle the case where the player is not found
		conn.WriteMessage(websocket.TextMessage, []byte("Player not found."))
		return
	}

	switch command {
	case "playcard":
		if len(parts) < 2 {
			// Handle invalid command format
			conn.WriteMessage(websocket.TextMessage, []byte("Invalid command format. Usage: playcard <card>"))
			return
		}
		//card := parts[1]
		// Call the PlayCard function for the player
		// err := playerPtr.PlayCard(card)
		// if err != nil {
		// 	conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		// } else {
		// 	conn.WriteMessage(websocket.TextMessage, []byte("Card played successfully."))
		// }
	case "showcards":
		// Call the ShowCards function for the player
		your_cards := playerPtr.CardInHand()
		conn.WriteMessage(websocket.TextMessage, []byte(your_cards))

	default:
		conn.WriteMessage(websocket.TextMessage, []byte("Chat msg"))
	}
}
