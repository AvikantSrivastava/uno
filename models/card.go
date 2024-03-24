package models

import (
	"uno/models/constants/color"
	"uno/models/constants/rank"
)

type Card struct {
	Rank  rank.Rank   `validate:"required"`
	Color color.Color `validate:"omitempty"`
}

func (c Card) Type() string {
	if c.Rank >= "0" && c.Rank <= "9" {
		return "number-card"
	} else if c.Rank == rank.WILD || c.Rank == rank.DRAW_4 {
		return "action-card-no-color"
	}
	return "action-card"
}

func (c Card) LogCard() string {
	return string(c.Rank) + " " + string(c.Color)
}

func (card Card) IsSameColor(otherCard Card) bool {
	// Implement color comparison logic
	return true // Replace with the actual implementation
}

// IsSameRank checks if two cards have the same rank
func (card Card) IsSameRank(otherCard Card) bool {
	// Implement rank comparison logic
	return true // Replace with the actual implementation
}
