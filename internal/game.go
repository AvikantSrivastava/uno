package internal

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
	"uno/models/commands"
	"uno/models/constants/color"
	"uno/models/constants/rank"
	"uno/models/dtos"
	"uno/models/game"

	"github.com/gorilla/websocket"
)

type Game struct {
	Players          []*game.Player
	GameDeck         *game.GameDeck
	DisposedGameDeck *game.GameDeck
	Room             *Room
	GameStarted      bool
	CurrentTurn      int
	GameDirection    bool
	ActivePlayer     *game.Player //pointer to active player
	mu               sync.Mutex
	TopCard          game.Card
	TopColor         color.Color
	GameFirstMove    bool
	Network          Network
}

func NewGame() *Game {
	gameDeck := game.NewGameDeck() //Initialised Game Deck
	disposedGameDeck := &game.GameDeck{
		Deck: &game.Deck{
			Cards: make([]game.Card, 0), // Initialize the Cards slice
		},
	}
	var (
		game = &Game{
			Players:          make([]*game.Player, 0),
			GameDeck:         gameDeck,
			DisposedGameDeck: disposedGameDeck,
			GameStarted:      false,
			GameDirection:    false,
			Network:          *NewNetwork(),
		}
	)
	game.SetTopCard(*gameDeck.GetStartCard())
	return game
}

func (g *Game) AddPlayer(player *game.Player) {
	g.mu.Lock()
	defer g.mu.Unlock()
	player.AddCards(g.GameDeck.Cut(7))
	g.Players = append(g.Players, player)
}

func (g *Game) NextTurn() {
	g.ActivePlayer.Drawn = false
	//check for Game winner
	if g.ActivePlayer.Deck.NumberOfCards() == 0 {
		g.declareWinner(g.ActivePlayer)
	}
	//Check for UNO
	if g.ActivePlayer.Deck.NumberOfCards() == 1 {
		g.checkforUNO(g.ActivePlayer)
	}

	integerDirection := convertDirectionToInteger(g.GameDirection)

	nextTurn := (g.CurrentTurn + integerDirection) % len(g.Players)
	if nextTurn < 0 {
		nextTurn += len(g.Players)
	}

	g.SetActivePlayer(nextTurn)
	g.Network.SendInfoMessage(g.ActivePlayer, "It is your turn.")
}
func (g *Game) Start() {
	// Start the first player's turn
	g.GameFirstMove = true
	g.SetActivePlayer(0)
	g.GameStarted = true
}

func (g *Game) PlayCard(p *game.Player, index int, newColor string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	card := p.Deck.Cards[index]

	switch {
	case card.Type() == "action-card-no-color":
		parsedColor, err := color.ParseColor(newColor)
		if err != nil {
			g.Network.SendInfoMessage(p, "Invalid color. Try again.")
			return
		}
		g.SetTopCard(card, parsedColor)
		g.DisposedGameDeck.AddCard(p.Deck.RemoveCard(index))
		g.Network.BroadcastInfoMessage(fmt.Sprintf("%s played %s and changed the color to %s", p.Name, card.LogCard(), newColor))

		if card.Rank == rank.DRAW_4 {
			g.PerformDrawAction(g.getNextPlayer(), 4)
			g.skipNextTurn()
		}
		g.NextTurn()
	case g.IsValidMove(card, p):
		if card.Type() == "action-card" {
			g.dealwithActionCards(card)
		}
		g.SetTopCard(card)
		g.Network.BroadcastInfoMessage(fmt.Sprintf("%s played %s", p.Name, card.LogCard()))
		g.DisposedGameDeck.AddCard(p.Deck.RemoveCard(index))
		g.NextTurn()

	default:
		g.Network.SendInfoMessage(p, "Invalid move. Wrong card or wrong player. Try again.")
	}
}


func (g *Game) SetActivePlayer(index int) {
	g.CurrentTurn = index
	g.ActivePlayer = g.Players[index]
}

func (g *Game) SetTopCard(card game.Card, color ...color.Color) {
	if card.Type() == "action-card-no-color" && len(color) > 0 {
		g.TopCard = card
		g.TopColor = color[0]
	} else {
		g.TopCard = card
		g.TopColor = card.Color
	}
}

func (g *Game) IsValidMove(playedCard game.Card, player *game.Player) bool {
	if player != g.ActivePlayer {
		return false
	}
	if g.TopCard.Type() == "action-card-no-color" {
		return playedCard.Color == g.TopColor
	}
	// If the played card matches the color or rank of the top card, it's a valid move
	return playedCard.IsSameColor(g.TopCard) || playedCard.IsSameRank(g.TopCard)
}

func (g *Game) PerformDrawAction(player *game.Player, card_count int) {
	cardsDrawn := g.GameDeck.Cut(card_count)
	player.AddCards(cardsDrawn)
	for _, card := range cardsDrawn {
		g.Network.SendInfoMessage(player, fmt.Sprintf("%s Drew %s", player.Name, card.LogCard()))
	}
	g.Network.SendInfoMessage(player, fmt.Sprintf("%s  Drew %d cards  ", player.Name, card_count))

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
		g.DisposedGameDeck.Deck.Cards = make([]game.Card, 0)
	}
}

// reverseGameDirection reverses the game direction
func (g *Game) reverseGameDirection() {
	g.GameDirection = !g.GameDirection
}

// skipNextTurn skips the next player's turn
func (g *Game) skipNextTurn() {
	nextPlayer := g.getNextPlayer()
	g.Network.SendInfoMessage(nextPlayer, "Your turn is SKIPPED")

	g.switchtoNextPlayer()
}

// declareWinner declares the winner of the game
func (g *Game) declareWinner(winner *game.Player) {
	for _, p := range g.Players {
		g.Network.SendInfoMessage(p, fmt.Sprintf("GAME OVER !  %sHAS WON THE GAME!! ,CLOSING CONNECTION ", winner.Name))
		g.Network.CloseConnection(p)
	}
	// Perform any necessary  end-game animation with bubbleTea
}
func (g *Game) checkforUNO(player *game.Player) {
	for _, p := range g.Players {
		g.Network.SendInfoMessage(p, fmt.Sprintf("UNO !!!! by %s ", player.Name))
	}
}

// getNextPlayer returns the next player based on the game direction
func (g *Game) getNextPlayer() *game.Player {
	integerDirection := convertDirectionToInteger(g.GameDirection)

	nextTurn := (g.CurrentTurn + integerDirection) % len(g.Players)
	if nextTurn < 0 {
		nextTurn += len(g.Players)

	}
	return g.Players[nextTurn]
}

func (g *Game) getAllPlayers() []string {
	var playerNames []string
	for _, player := range g.Players {
		playerNames = append(playerNames, player.Name)
	}
	return playerNames
}

func (g *Game) dealwithActionCards(card game.Card) {
	cardType := card.Rank
	switch cardType {
	case "skip":
		g.skipNextTurn()
	case "draw_2":
		g.PerformDrawAction(g.getNextPlayer(), 2)
		g.skipNextTurn()
	case "reverse":
		log.Println("game direction will be reversed")
		log.Println("Current Turn: ", g.ActivePlayer.Name)
		g.reverseGameDirection()
		log.Println("Game direction reversed now")
	default:
		// Handle unexpected card types here, e.g., log an error
		fmt.Println("Unexpected card type:", cardType)
	}
}

func (g *Game) switchtoNextPlayer() {
	integerDirection := convertDirectionToInteger(g.GameDirection)
	g.CurrentTurn = (g.CurrentTurn + integerDirection) % len(g.Players)
	if g.CurrentTurn < 0 {
		g.CurrentTurn += len(g.Players)

	}
}

func (g *Game) HandleCommand(data []byte, player *game.Player) {
	cmd, err := commands.DeserializeCommand(data)
	if err != nil {
		log.Fatalf("Failed to deserialize command: %v", err)
	}

	switch c := cmd.(type) {
	case *commands.SyncCommand:
		g.SyncPlayer(player)
	case *commands.PlayCardCommand:
		if g.ActivePlayer == player {
			g.PlayCard(player, c.CardIndex, c.NewColor)
		}
		g.SyncAllPlayers()
	case *commands.DrawCardComamnd:
		if g.ActivePlayer == player && player.Drawn == false {
			g.PerformDrawAction(player, 1)
			player.Drawn = true
		}
		// Check if the player has a playable card (including the drawn one)
		if player.HasPlayableCard(g.TopCard, g.TopColor) {
			player.Drawn = false //To make sure After getting playable card the turn isn't skipped
			g.Network.SendInfoMessage(player, "It is still your turn.")
		} else {
			g.NextTurn() // No playable card, move to the next player's turn
		}
		g.SyncAllPlayers()
	default:
		log.Printf("Unknown command type: %T", c)

	}

}

func (g *Game) SyncPlayer(p *game.Player) {
	if p == nil {
		log.Printf("Cannot sync a nil player")
		return
	}

	activePlayer := g.ActivePlayer
	if activePlayer == nil {
		log.Printf("ActivePlayer is nil; cannot sync")
		return
	}

	if activePlayer.Name == "" {
		log.Printf("ActivePlayer's Name is empty; cannot sync")
		return
	}

	dto := dtos.SyncDTO{
		Player: *p,
		Game: dtos.GameState{
			TopCard:  g.TopCard,
			TopColor: g.TopColor,
			Turn:     activePlayer.Name,
			Reverse:  g.GameDirection,
		},
		Room: dtos.RoomState{
			Players:    g.getAllPlayers(),
			RoomId:     g.Room.id,
			MaxPlayers: g.Room.maxPlayers,
		},
	}

	conn, ok := g.Network.clients[*p]
	if !ok {
		log.Printf("SyncPlayer Error:Player %s not found in network clients", p.Name)
		//log.Printf("Top card is %s ", g.TopCard)
		return
	}

	lock, ok := g.Network.locks[*p]
	if !ok {
		log.Printf("No mutex found for player %s", p.Name)
		return
	}

	lock.Lock()
	defer lock.Unlock()

	err := conn.WriteMessage(websocket.TextMessage, dto.Serialize())
	if err != nil {
		log.Printf("Failed to sync player %s: %v", p.Name, err)
	}
}

func (g *Game) SyncAllPlayers() {
	var wg sync.WaitGroup
	for _, playerPtr := range g.Players {
		wg.Add(1)

		go func(player *game.Player) {
			defer wg.Done()
			g.SyncPlayer(player)
		}(playerPtr)
	}

	wg.Wait()
}
