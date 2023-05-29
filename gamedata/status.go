package gamedata

import (
	"encoding/json"
	"log"

	"github.com/s22678/statki/connect"
)

var (
	gameEndpoint = "/api/game"
)

type Status struct {
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

func GetStatus(c *connect.Connection) (*Status, error) {
	gd := &Status{}
	body, err := c.GameAPIConnection("GET", gameEndpoint, nil)
	if err != nil {
		log.Println("Status:", string(body))
		return nil, err
	}

	err = json.Unmarshal(body, &gd)
	if err != nil {
		return nil, err
	}

	return gd, err
}
