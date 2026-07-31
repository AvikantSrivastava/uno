package internal

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
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
	GameEnded        bool
	CurrentTurn      int
	GameDirection    bool
	ActivePlayer     *game.Player
	mu               sync.RWMutex
	TopCard          game.Card
	TopColor         color.Color
	GameFirstMove    bool
	Network          Network
}

func NewGame() *Game {
	gameDeck := game.NewGameDeck()
	disposedGameDeck := &game.GameDeck{
		Deck: &game.Deck{
			Cards: make([]game.Card, 0),
		},
	}
	g := &Game{
		Players:          make([]*game.Player, 0),
		GameDeck:         gameDeck,
		DisposedGameDeck: disposedGameDeck,
		GameStarted:      false,
		GameEnded:        false,
		GameDirection:    false,
		Network:          *NewNetwork(),
	}
	if startCard := gameDeck.GetStartCard(); startCard != nil {
		g.SetTopCard(*startCard)
	}
	return g
}

func (g *Game) AddPlayer(player *game.Player) {
	g.mu.Lock()
	defer g.mu.Unlock()
	cards := g.drawCards(7)
	player.AddCards(cards)
	g.Players = append(g.Players, player)
}

func (g *Game) RemovePlayer(player *game.Player) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i, p := range g.Players {
		if p == player {
			g.Players = append(g.Players[:i], g.Players[i+1:]...)
			break
		}
	}
}

// drawCards draws cards from the deck, reshuffling discard pile if needed
// Must be called with g.mu held
func (g *Game) drawCards(count int) []game.Card {
	if len(g.GameDeck.Deck.Cards) < count {
		g.shuffleDiscardPileToDeck()
	}
	return g.GameDeck.Cut(count)
}

// shuffleDiscardPileToDeck reshuffles the discard pile back into the deck
// Must be called with g.mu held
func (g *Game) shuffleDiscardPileToDeck() {
	if len(g.DisposedGameDeck.Deck.Cards) == 0 {
		return
	}

	rand.Shuffle(len(g.DisposedGameDeck.Deck.Cards), func(i, j int) {
		g.DisposedGameDeck.Deck.Cards[i], g.DisposedGameDeck.Deck.Cards[j] = g.DisposedGameDeck.Deck.Cards[j], g.DisposedGameDeck.Deck.Cards[i]
	})

	g.GameDeck.Deck.Cards = append(g.GameDeck.Deck.Cards, g.DisposedGameDeck.Deck.Cards...)
	g.GameDeck.Deck.Counter = len(g.GameDeck.Deck.Cards)
	g.DisposedGameDeck.Deck.Cards = make([]game.Card, 0)
	g.DisposedGameDeck.Deck.Counter = 0

	g.Network.BroadcastInfoMessage("Shuffling the deck...")
}

func (g *Game) NextTurn() {
	if g.ActivePlayer == nil || g.GameEnded {
		return
	}

	g.ActivePlayer.Drawn = false

	// Check for game winner
	if g.ActivePlayer.Deck.NumberOfCards() == 0 {
		g.declareWinner(g.ActivePlayer)
		return
	}

	// Check for UNO
	if g.ActivePlayer.Deck.NumberOfCards() == 1 {
		g.checkforUNO(g.ActivePlayer)
	}

	if len(g.Players) == 0 {
		return
	}

	integerDirection := convertDirectionToInteger(g.GameDirection)
	nextTurn := (g.CurrentTurn + integerDirection) % len(g.Players)
	if nextTurn < 0 {
		nextTurn += len(g.Players)
	}

	g.SetActivePlayer(nextTurn)
	if g.ActivePlayer != nil {
		g.Network.SendInfoMessage(g.ActivePlayer, "It is your turn.")
	}
}
func (g *Game) Start() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if len(g.Players) == 0 {
		return
	}
	g.GameFirstMove = true
	g.SetActivePlayer(0)
	g.GameStarted = true
}

func (g *Game) PlayCard(p *game.Player, index int, newColor string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.GameEnded {
		return
	}

	// Bounds check for card index
	if index < 0 || index >= len(p.Deck.Cards) {
		g.Network.SendInfoMessage(p, "Hmm, that card doesn't exist")
		return
	}

	card := p.Deck.Cards[index]

	switch {
	case card.Type() == "action-card-no-color":
		parsedColor, err := color.ParseColor(newColor)
		if err != nil {
			g.Network.SendInfoMessage(p, "Pick a valid color: red, blue, green, or yellow")
			return
		}
		g.SetTopCard(card, parsedColor)
		g.DisposedGameDeck.AddCard(p.Deck.RemoveCard(index))
		g.Network.BroadcastInfoMessage(fmt.Sprintf("%s played %s - Color is now %s", p.Name, card.LogCard(), newColor))

		if card.Rank == rank.DRAW_4 {
			nextPlayer := g.getNextPlayer()
			if nextPlayer != nil {
				g.performDrawActionLocked(nextPlayer, 4)
				g.skipNextTurn()
			}
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
		g.Network.SendInfoMessage(p, "Can't play that card right now")
	}
}

func (g *Game) SetActivePlayer(index int) {
	if index < 0 || index >= len(g.Players) {
		return
	}
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

func (g *Game) PerformDrawAction(player *game.Player, cardCount int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.performDrawActionLocked(player, cardCount)
}

// performDrawActionLocked performs draw action - must be called with g.mu held
func (g *Game) performDrawActionLocked(player *game.Player, cardCount int) {
	if player == nil {
		return
	}

	cardsDrawn := g.drawCards(cardCount)
	if len(cardsDrawn) == 0 {
		g.Network.SendInfoMessage(player, "No cards left in the deck!")
		return
	}

	player.AddCards(cardsDrawn)
	if len(cardsDrawn) == 1 {
		g.Network.SendInfoMessage(player, fmt.Sprintf("You drew a %s", cardsDrawn[0].LogCard()))
	} else {
		g.Network.SendInfoMessage(player, fmt.Sprintf("You picked up %d cards", len(cardsDrawn)))
	}
	g.Network.SendInfoMessage(player, fmt.Sprintf("%s drew %d card(s)", player.Name, len(cardsDrawn)))
}

// reverseGameDirection reverses the game direction
func (g *Game) reverseGameDirection() {
	g.GameDirection = !g.GameDirection
}

// skipNextTurn skips the next player's turn
func (g *Game) skipNextTurn() {
	nextPlayer := g.getNextPlayer()
	if nextPlayer != nil {
		g.Network.SendInfoMessage(nextPlayer, "Oops! Your turn was skipped")
	}
	g.switchtoNextPlayer()
}

// declareWinner declares the winner of the game
func (g *Game) declareWinner(winner *game.Player) {
	g.GameEnded = true

	for _, p := range g.Players {
		g.Network.SendInfoMessage(p, fmt.Sprintf("%s HAS WON THE GAME!!!!", winner.Name))
	}
	for _, p := range g.Players {
		g.Network.SendInfoMessage(p, fmt.Sprintf("GAME OVER %s, CLOSING CONNECTION", winner.Name))
		g.Network.CloseConnection(p)
	}

	// Cleanup the room
	if g.Room != nil {
		go g.Room.cleanup()
	}
}
func (g *Game) checkforUNO(player *game.Player) {
	g.Network.BroadcastInfoMessage(fmt.Sprintf("UNO! %s has one card left!", player.Name))
}

// getNextPlayer returns the next player based on the game direction
func (g *Game) getNextPlayer() *game.Player {
	if len(g.Players) == 0 {
		return nil
	}

	integerDirection := convertDirectionToInteger(g.GameDirection)
	nextTurn := (g.CurrentTurn + integerDirection) % len(g.Players)
	if nextTurn < 0 {
		nextTurn += len(g.Players)
	}

	if nextTurn < 0 || nextTurn >= len(g.Players) {
		return nil
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
		nextPlayer := g.getNextPlayer()
		if nextPlayer != nil {
			g.performDrawActionLocked(nextPlayer, 2)
			g.skipNextTurn()
		}
	case "reverse":
		g.reverseGameDirection()
	default:
		log.Printf("Unexpected card type: %s", cardType)
	}
}

func (g *Game) switchtoNextPlayer() {
	if len(g.Players) == 0 {
		return
	}
	integerDirection := convertDirectionToInteger(g.GameDirection)
	g.CurrentTurn = (g.CurrentTurn + integerDirection) % len(g.Players)
	if g.CurrentTurn < 0 {
		g.CurrentTurn += len(g.Players)
	}
}

func (g *Game) HandleCommand(data []byte, player *game.Player) {
	cmd, err := commands.DeserializeCommand(data)
	if err != nil {
		log.Printf("Failed to deserialize command from player %s: %v", player.Name, err)
		g.Network.SendInfoMessage(player, "Something went wrong, try again")
		return
	}

	g.mu.Lock()
	activePlayer := g.ActivePlayer
	topCard := g.TopCard
	gameEnded := g.GameEnded
	g.mu.Unlock()

	if gameEnded {
		return
	}

	switch c := cmd.(type) {
	case *commands.SyncCommand:
		g.SyncPlayer(player)
	case *commands.PlayCardCommand:
		if activePlayer == player {
			g.PlayCard(player, c.CardIndex, c.NewColor)
		} else {
			g.Network.SendInfoMessage(player, "Hold on! It's not your turn yet")
		}
		g.SyncAllPlayers()
	case *commands.DrawCardCommand:
		g.mu.Lock()
		if g.ActivePlayer == player && !player.Drawn {
			g.performDrawActionLocked(player, 1)
			player.Drawn = true
		}
		if g.ActivePlayer != nil && !g.ActivePlayer.HasPlayableCard(topCard) {
			g.NextTurn()
		}
		g.mu.Unlock()
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

	g.mu.RLock()
	activePlayer := g.ActivePlayer
	if activePlayer == nil {
		g.mu.RUnlock()
		log.Printf("ActivePlayer is nil; cannot sync")
		return
	}

	if g.Room == nil {
		g.mu.RUnlock()
		log.Printf("Room is nil; cannot sync")
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
	g.mu.RUnlock()

	g.Network.mu.RLock()
	conn, connOk := g.Network.clients[*p]
	lock, lockOk := g.Network.locks[*p]
	g.Network.mu.RUnlock()

	if !connOk {
		log.Printf("Player %s not found in network clients", p.Name)
		return
	}

	if !lockOk {
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
	g.mu.RLock()
	players := make([]*game.Player, len(g.Players))
	copy(players, g.Players)
	g.mu.RUnlock()

	var wg sync.WaitGroup
	for _, playerPtr := range players {
		wg.Add(1)
		go func(player *game.Player) {
			defer wg.Done()
			g.SyncPlayer(player)
		}(playerPtr)
	}
	wg.Wait()
}
