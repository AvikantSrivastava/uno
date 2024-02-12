package models

import (
	"github.com/AvikantSrivastava/uno/models/constants/color"
	"github.com/AvikantSrivastava/uno/models/constants/rank"
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

func (bottomCard Card) ValidCard(topCard Card) bool {
	if bottomCard.Type() == "action-bottomCard-no-color" {
		return true
	} else if bottomCard.Color == topCard.Color || bottomCard.Rank == topCard.Rank {
		return true
	}
	return false
}
