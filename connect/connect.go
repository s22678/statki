package connect

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

const (
	game_endpoint = "/api/game"
)

// func InitGame() error
// func Board() ([]string, error)
// func Status() (*StatusResponse, error)
// func Fire(coord string) (string, error)

type Connection struct {
	token string
	Url   string
}

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

var (
	gameConnectionInit = map[string]interface{}{
		"coords":      nil,
		"desc":        "",
		"nick":        "",
		"target_nick": "",
		"wpbot":       true,
	}
)

func (s *Connection) InitGame() error {
	b, err := json.Marshal(gameConnectionInit)
	if err != nil {
		log.Println(err)
	}

	reader := bytes.NewReader(b)
	log.Println(string(b))
	resp, err := http.Post(s.Url+game_endpoint, "application/json", reader)
	if err != nil {
		log.Println(resp)
		log.Println(err)
	}
	s.token = resp.Header.Get("X-Auth-Token")
	fmt.Println(resp.Header.Get("X-Auth-Token"))
	return err
}

func (s *Connection) Status() (*StatusResponse, error) {
	sr := StatusResponse{}
	client := http.Client{}
	req, err := http.NewRequest("GET", s.Url+game_endpoint, nil)
	if err != nil {
		log.Println(req)
		log.Println(err)
	}
	req.Header.Set("X-Auth-Token", s.token)
	r, err := client.Do(req)
	if err != nil {
		log.Println(req)
		log.Println(err)
	}
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
	}

	err = json.Unmarshal(body, &sr)
	if err != nil {
		log.Println(err)
	}

	log.Println(string(body))

	return &sr, err
}
