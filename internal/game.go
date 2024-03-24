package internal

import (
	"fmt"
	"strconv"
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
	GameFirstMove    bool
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
	fmt.Printf("It's your turn, %s! ", g.ActivePlayer.Name)
	//g.DisposedGameDeck

	//g.ActivePlayer.Send()
}
func (g *Game) Start() {
	// g.mu.Lock()
	// defer g.mu.Unlock()

	// Start the first player's turn
	g.CurrentTurn = 0
	g.GameDirection = true
	g.GameFirstMove = true
	g.ActivePlayer = g.Players[g.CurrentTurn] //g.Players is already a pointer
	for _, p := range g.Players {

		err := p.Send(fmt.Sprintf("It's %s's turn.Please play your turn.\n", g.ActivePlayer.Name))
		if err != nil {
			// Handle the error
			fmt.Printf("Error sending message to player %s: %v\n", p.Name, err)
		}
	}

}

func (g *Game) PlayCard(player *models.Player, cardIdx int) { // Reason why it 's Game func because of isValidMove ,can check gametopcard state and confirm
	g.mu.Lock()
	defer g.mu.Unlock()
	card := player.Deck.Cards[cardIdx]
	// Check if the card is playable
	//Might remove this isValidMove to another package
	if g.IsValidMove(card, player) {
		// Perform game logic for playing the card
		// Update game state, check for UNO, etc.

		// Notify all players about the played card
		for _, p := range g.Players {
			p.Send(fmt.Sprintf("%s played %s", player.Name, card.LogCard()))
		}

		// Check if the player has won\
		//Add Card to disposed Deck
		DisposedCard := player.Deck.RemoveCard(cardIdx)
		g.DisposedGameDeck.AddCard(DisposedCard)
		g.GameTopCard = &DisposedCard //  assignment to a pointer //Set Game top card
		// Move to the next turn
		g.NextTurn()
	} else {
		// Notify the player that the move is invalid
		player.Send("Invalid move. Try again. or Wrong player")
	}
}
func (g *Game) IsValidMove(playedcard models.Card, player *models.Player) bool {
	if g.GameFirstMove {
		g.GameFirstMove = false //First card will not check deck's top card
		return true
	}
	GameTopCard := g.GameTopCard
	if playedcard.IsSameColor(*GameTopCard) || playedcard.IsSameRank(*GameTopCard) {
		return true
	}
	if player != g.ActivePlayer {
		return false
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
		cardidx, _ := strconv.Atoi(parts[1])
		// Call the PlayCard function for the player
		g.PlayCard(playerPtr, cardidx)

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
