package dtos

type PongDTO struct {
	ConnectedPlayers int `json:"connected_players"`
}

func (dto PongDTO) Serialize() []byte {
	return Serialize(dto, "pong")
}
