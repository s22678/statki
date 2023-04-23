package app

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"reflect"

	"github.com/fatih/color"
	gui "github.com/grupawp/warships-lightgui"
	"github.com/s22678/statki/connect"
)

const (
	maxConnectionAttempts = 10
	boardEndpoint         = "/api/game/board"
)

type GameBoard struct {
	Board []string "json:board"
}

type Application struct {
	Con   connect.Connection
	GB    GameBoard
	board *gui.Board
}

func downloadBoard(a *Application) {
	client := http.Client{}
	req, err := http.NewRequest("GET", a.Con.Url+boardEndpoint, nil)
	if err != nil {
		log.Println(req, err)
	}
	req.Header.Set("X-Auth-Token", a.Con.Token)
	r, err := client.Do(req)
	if err != nil {
		log.Println(req, err)
	}
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
	}

	err = json.Unmarshal(body, &a.GB)
	if err != nil {
		log.Println(err)
	}
	log.Println(string(body))
}

func initGameBoard(a *Application) *gui.Board {
	board := gui.New(
		gui.ConfigParams().
			HitChar('#').
			HitColor(color.FgRed).
			BorderColor(color.BgRed).
			RulerTextColor(color.BgYellow).
			NewConfig(),
	)
	board.Import(a.GB.Board)
	return board
}

func (a *Application) Board() {
	if reflect.DeepEqual(a.GB, GameBoard{}) {
		downloadBoard(a)
		a.board = initGameBoard(a)
	}
	a.board.Display()
	// a.c.InitGame(nil, "", "", false)
}
