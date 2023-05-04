package connect

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
)

const (
	ServerUrl    = "https://go-pjatk-server.fly.dev"
	gameEndpoint = "/api/game"
)

var (
	GameConnectionData = map[string]interface{}{
		"coords":      nil,
		"desc":        "",
		"nick":        "",
		"target_nick": "",
		"wpbot":       true,
	}

	ErrEmptyTokenException      = errors.New("the connection token is empty. re-initialize connection to the game server")
	ErrSessionNotFoundException = errors.New("the game is over")
)

type Connection struct {
	token string
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

func (connection *Connection) GetToken() (string, error) {
	if len(connection.token) > 0 {
		return connection.token, nil
	}

	return "", ErrEmptyTokenException
}

func (connection *Connection) InitGame(playWithBot bool, enemyNick string, playerNick string, playerDescription string) error {
	GameConnectionData["wpbot"] = playWithBot
	GameConnectionData["desc"] = playerDescription
	GameConnectionData["target_nick"] = enemyNick
	GameConnectionData["nick"] = playerNick
	b, err := json.Marshal(GameConnectionData)
	if err != nil {
		log.Println(err)
	}

	reader := bytes.NewReader(b)
	log.Println(string(b))
	resp, err := http.Post(ServerUrl+gameEndpoint, "application/json", reader)
	if err != nil {
		log.Println(resp)
		log.Println(err)
	}
	connection.token = resp.Header.Get("X-Auth-Token")
	log.Println(resp.Header.Get("X-Auth-Token"))
	return err
}

func (connection *Connection) GameAPIConnection(HTTPMethod string, endpoint string, reqBody io.Reader) ([]byte, error) {
	client := http.Client{}
	req, err := http.NewRequest(HTTPMethod, ServerUrl+endpoint, reqBody)
	if err != nil {
		log.Println(req, err)
		return nil, err
	}

	if connection.token != "" {
		req.Header.Set("X-Auth-Token", connection.token)
	}
	r, err := client.Do(req)
	if err != nil {
		log.Println(req, err)
		return nil, err
	}
	log.Println(HTTPMethod, endpoint, r.Status)
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if r.StatusCode == 403 {
		return body, ErrSessionNotFoundException
	}

	// TODO - should this method return a []byte?
	return body, nil
}
