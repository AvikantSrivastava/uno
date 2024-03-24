package internal

import (
	"fmt"
	"strings"
	"sync"
	"uno/models"

	"github.com/gorilla/websocket"
)

type Game struct {
	Players          []*models.Player
	GameDeck         *models.GameDeck
	DisposedGameDeck *models.GameDeck
	CurrentTurn      int
	GameDirection    bool
	ActivePlayer     *models.Player //pointer to active player
	mu               sync.Mutex
	GameTopCard      *models.Card //TopCard of Playing Game
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
	disposedGameDeck := &models.GameDeck{
		Deck: &models.Deck{
			Cards: make([]models.Card, 0), // Initialize the Cards slice
		},
	}

	game := &Game{
		Players:          players,
		GameDeck:         gameDeck,
		DisposedGameDeck: disposedGameDeck,
	}
	return game
}

func (g *Game) NextTurn() {
	// g.ActivePlayer.mu.Lock()
	// defer g.ActivePlayer.mu.Unlock()

	if g.ActivePlayer != nil {

		g.ActivePlayer.Send("Your turn is over.")
	}

	integerDirection := convertDirectionToInteger(g.GameDirection)
	g.CurrentTurn = (g.CurrentTurn + integerDirection) % len(g.Players)

	g.ActivePlayer = g.Players[g.CurrentTurn]
	fmt.Printf("It's your turn, %s! and turn %d", g.ActivePlayer.Name, g.CurrentTurn)
	//g.DisposedGameDeck

	//g.ActivePlayer.Send()
}
func (g *Game) Start() {
	// g.mu.Lock()
	// defer g.mu.Unlock()

	// Start the first player's turn
	g.CurrentTurn = 0
	g.GameDirection = true

	g.ActivePlayer = g.Players[g.CurrentTurn] //g.Players is already a pointer
	//g.ActivePlayer.CardInHand()
	//Commenting start
	//PlayCard then next turn
	g.NextTurn()

}

func (g *Game) PlayCard(player *models.Player, card models.Card) { // Reason why it 's Game func because of isValidMove ,can check gametopcard state and confirm
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
		//g.PlayCard(playerPtr, card)
		conn.WriteMessage(websocket.TextMessage, []byte("Card played successfully."))

	case "showcards":
		// Call the ShowCards function for the player
		your_cards := playerPtr.CardInHand()
		conn.WriteMessage(websocket.TextMessage, []byte(your_cards))
	case "topcard":
		conn.WriteMessage(websocket.TextMessage, []byte(g.GameTopCard.LogCard()))
	default:
		conn.WriteMessage(websocket.TextMessage, []byte("Chat msg"))
	}
}
