package internal

import (
	"uno/models"

	"golang.org/x/exp/slices"
)

func ShouldBroadcast(msg string) bool {
	privateCommands := []string{"showcards"}

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
