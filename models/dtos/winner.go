package dtos

type WinnerDTO struct {
	WinnerName string `json:"winner_name"`
	IsWinner   bool   `json:"is_winner"`
}

func (dto WinnerDTO) Serialize() []byte {
	return Serialize(dto, "game_over")
}
