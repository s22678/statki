package gamedata

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/s22678/statki/connect"
)

const (
	listEndpoint = "/api/game/list"
)

type Player struct {
	Game_status string `json:"game_status,omitempty"`
	Nick        string `json:"nick,omitempty"`
}

func ListPlayers() ([]Player, error) {
	players := []Player{}
	client := http.Client{}
	req, err := http.NewRequest("GET", connect.ServerUrl+listEndpoint, nil)
	if err != nil {
		log.Println(req, err)
	}

	r, err := client.Do(req)
	if err != nil {
		log.Println(req, err)
	}
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
	}

	err = json.Unmarshal(body, &players)
	if err != nil {
		log.Println(err)
	}

	log.Println("lista graczy: ", string(body))

	return players, err
}
