package gamedata

import (
	"encoding/json"
	"log"

	"github.com/s22678/statki/connect"
)

var (
	descEndpoint = "/api/game/desc"
)

type PlayerInfo struct {
	Nick     string `json:"nick,omitempty"`
	Desc     string `json:"desc,omitempty"`
	Opponent string `json:"opponent,omitempty"`
	Opp_desc string `json:"opp_desc,omitempty"`
}

func GetPlayerInfo(c *connect.Connection) (*PlayerInfo, error) {
	pi := &PlayerInfo{}
	body, err := c.GameAPIConnection("GET", descEndpoint, nil)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	err = json.Unmarshal(body, &pi)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	log.Println("Player Info:", string(body))

	return pi, nil
}
