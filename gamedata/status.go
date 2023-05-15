package gamedata

import (
	"encoding/json"
	"log"

	"github.com/s22678/statki/connect"
)

var (
	gameEndpoint = "/api/game"
)

type StatusResponse struct {
	Desc             string   `json:"desc,omitempty"`
	Game_status      string   `json:"game_status,omitempty"`
	Last_game_status string   `json:"last_game_status,omitempty"`
	Nick             string   `json:"nick,omitempty"`
	Opp_desc         string   `json:"opp_desc,omitempty"`
	Opp_shots        []string `json:"opp_shots,omitempty"`
	Opponent         string   `json:"opponent,omitempty"`
	Should_fire      bool     `json:"should_fire,omitempty"`
	Timer            int      `json:"timer,omitempty"`
}

func (sr *StatusResponse) GetOpponent() string {
	return sr.Opponent
}

func (g *StatusResponse) GetDescription() string {
	return g.Desc
}

func GetStatus(c *connect.Connection) (*StatusResponse, error) {
	gd := &StatusResponse{}
	body, err := c.GameAPIConnection("GET", gameEndpoint, nil)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &gd)
	if err != nil {
		return nil, err
	}

	log.Println("Status:", string(body))

	return gd, err
}
