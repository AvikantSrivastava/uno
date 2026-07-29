package commands

import (
	"fmt"
	"sync"
)

type CommandFactory func() interface{}

// CommandRegistry maintains a map of command types to their respective factories.
var CommandRegistry = struct {
	mu      sync.RWMutex
	entries map[string]CommandFactory
}{
	entries: make(map[string]CommandFactory),
}

// RegisterCommand registers a new command type with its factory function.
func RegisterCommand(commandType string, factory CommandFactory) {
	CommandRegistry.mu.Lock()
	defer CommandRegistry.mu.Unlock()
	CommandRegistry.entries[commandType] = factory
}

// GetCommandFactory retrieves the factory function for a given command type.
func GetCommandFactory(commandType string) (CommandFactory, error) {
	CommandRegistry.mu.RLock()
	defer CommandRegistry.mu.RUnlock()
	factory, exists := CommandRegistry.entries[commandType]
	if !exists {
		return nil, fmt.Errorf("no factory registered for command type: %s", commandType)
	}
	return factory, nil
}

func init() {
	RegisterCommand("SYNC_GAME_STATE", func() interface{} { return &SyncCommand{} })
	RegisterCommand("PLAY_CARD", func() interface{} { return &PlayCardCommand{} })
	RegisterCommand("DRAW_CARD", func() interface{} { return &DrawCardCommand{} })
}
