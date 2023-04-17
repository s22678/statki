package app

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/fatih/color"
	gui "github.com/grupawp/warships-lightgui"
	"github.com/s22678/statki/connect"
)

const (
	maxConnectionAttempts = 10
	boardEndpoint         = "/api/game/board"
)

type Board struct {
	Board []string "json:board"
}

type Application struct {
	Con connect.Connection
}

func (a *Application) Board() {
	b := Board{}
	client := http.Client{}
	req, err := http.NewRequest("GET", a.Con.Url+boardEndpoint, nil)
	if err != nil {
		log.Println(req)
		log.Println(err)
	}
	req.Header.Set("X-Auth-Token", a.Con.Token)
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

	err = json.Unmarshal(body, &b)
	if err != nil {
		log.Println(err)
	}

	log.Println(string(body))

	board := gui.New(
		gui.ConfigParams().
			HitChar('#').
			HitColor(color.FgRed).
			BorderColor(color.BgRed).
			RulerTextColor(color.BgYellow).
			NewConfig(),
	)
	board.Import(b.Board)
	board.Display()
	// a.c.InitGame(nil, "", "", false)
}
