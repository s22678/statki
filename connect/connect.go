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
	ErrEmptyTokenException      = errors.New("the connection token is empty. re-initialize connection to the game server")
	ErrSessionNotFoundException = errors.New("the game is over")
	ErrUnauthorizedException    = errors.New("error creating a new game, unauthorized")
)

type ConnectionData struct {
	Coords      []string `json:"coords,omitempty"`
	Desc        string   `json:"desc,omitempty"`
	Nick        string   `json:"nick,omitempty"`
	Target_nick string   `json:"target_nick,omitempty"`
	Wpbot       bool     `json:"wpbot,omitempty"`
}

type Connection struct {
	Data  ConnectionData
	token string
}

func (connection *Connection) GetToken() (string, error) {
	if len(connection.token) > 0 {
		return connection.token, nil
	}

	return "", ErrEmptyTokenException
}

func (connection *Connection) InitGame(playWithBot bool, enemyNick string, playerNick string, playerDescription string, playerShipsCoords []string) error {

	connection.Data.Coords = playerShipsCoords
	connection.Data.Desc = playerDescription
	connection.Data.Nick = playerNick
	connection.Data.Target_nick = enemyNick
	connection.Data.Wpbot = playWithBot
	b, err := json.Marshal(connection.Data)
	if err != nil {
		log.Println(err)
	}

	reader := bytes.NewReader(b)
	log.Println(string(b))
	resp, err := http.Post(ServerUrl+gameEndpoint, "application/json", reader)
	if err != nil {
		log.Println(resp)
		log.Println(err)
		return err
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

	if r.StatusCode == 401 {
		return body, ErrUnauthorizedException
	}

	// TODO - should this method return a []byte?
	return body, nil
}
