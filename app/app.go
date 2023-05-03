package app

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/s22678/statki/connect"
)

const (
	Left                  = 0 // indicates player's board (on the left side)
	Right                 = 1 // indicates enemy's board (on the right side)
	maxConnectionAttempts = 10
	boardEndpoint         = "/api/game/board"
	fireEndpoint          = "/api/game/fire"
	descEndpoint          = "/api/game/desc"
)

type Application struct {
	Con connect.Connection
}

func (a *Application) Fire(coord string) (string, error) {
	fireResponse := map[string]string{
		"result": "",
	}
	shot := make(map[string]interface{})
	shot["coord"] = coord
	b, err := json.Marshal(shot)
	if err != nil {
		log.Println(err)
	}

	reader := bytes.NewReader(b)
	log.Println(string(b))
	request, err := http.NewRequest("POST", connect.ServerUrl+fireEndpoint, reader)
	if err != nil {
		log.Println(err)
		return "", nil
	}
	token, _ := a.Con.GetToken()
	request.Header.Add("X-Auth-Token", token)
	client := &http.Client{}
	res, err := client.Do(request)
	if err != nil {
		log.Println(request, err)
		return "", err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Println(err)
		return "", nil
	}
	log.Println("Fire response body", string(body))
	err = json.Unmarshal(body, &fireResponse)
	if err != nil {
		log.Println(err)
		return "", err
	}
	return fireResponse["result"], nil
}

func (a *Application) GetDescritpion() (*connect.StatusResponse, error) {
	sr := connect.StatusResponse{}
	client := http.Client{}
	req, err := http.NewRequest("GET", connect.ServerUrl+descEndpoint, nil)
	if err != nil {
		log.Println(req, err)
	}
	token, _ := a.Con.GetToken()
	req.Header.Set("X-Auth-Token", token)
	r, err := client.Do(req)
	if err != nil {
		log.Println(req, err)
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

	log.Println("App description: ", string(body))

	return &sr, err
}

func (a *Application) QuitGame() error {
	_, err := a.Con.GameAPIConnection("DELETE", "/api/game/abandon")
	if err != nil {
		return err
	}
	return nil
}
