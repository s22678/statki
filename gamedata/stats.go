package gamedata

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/s22678/statki/connect"
)

const (
	statsEndpoint = "/api/game/stats"
)

type PlayerStats struct {
	Nick   string `json:"nick"`
	Games  int    `json:"games"`
	Wins   int    `json:"wins"`
	Rank   int    `json:"rank"`
	Points int    `json:"points"`
}

func GetAllPlayersStats() ([]PlayerStats, error) {
	stats := make(map[string][]PlayerStats)
	client := http.Client{}
	req, err := http.NewRequest("GET", connect.ServerUrl+statsEndpoint, nil)
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

	err = json.Unmarshal(body, &stats)
	if err != nil {
		log.Println(err)
	}

	log.Println("ranking zawodników: ", string(body))

	return stats["stats"], err
}

func GetOnePlayerStats(nick string) (PlayerStats, error) {
	stats := make(map[string]PlayerStats)
	client := http.Client{}
	req, err := http.NewRequest("GET", connect.ServerUrl+statsEndpoint+"/"+nick, nil)
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

	err = json.Unmarshal(body, &stats)
	if err != nil {
		log.Println(err)
	}

	log.Println("ranking zawodników: ", string(body))

	return stats["stats"], err
}
