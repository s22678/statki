package app

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/fatih/color"
	gui "github.com/grupawp/warships-lightgui/v2"
	"github.com/s22678/statki/connect"
)

const (
	maxConnectionAttempts = 10
	boardEndpoint         = "/api/game/board"
	fireEndpoint          = "/api/game/fire"
	Left                  = iota // indicates player's board (on the left side)
	Right                        // indicates enemy's board (on the right side)
)

type GameBoard struct {
	Board []string "json:board"
}

type Application struct {
	Con   connect.Connection
	board *gui.Board
}

func (a *Application) downloadBoard() (*GameBoard, error) {
	gb := &GameBoard{}
	client := http.Client{}
	req, err := http.NewRequest("GET", a.Con.Url+boardEndpoint, nil)
	if err != nil {
		log.Println(req, err)
		return nil, err
	}

	req.Header.Set("X-Auth-Token", a.Con.Token)
	r, err := client.Do(req)
	if err != nil {
		log.Println(req, err)
		return nil, err
	}

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	err = json.Unmarshal(body, gb)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	log.Println(string(body))
	return gb, nil
}

func (a *Application) initGameBoard(gameBoard []string) {
	cfg := gui.NewConfig()
	cfg.HitChar = '#'
	cfg.HitColor = color.FgRed
	cfg.BorderColor = color.BgRed
	cfg.RulerTextColor = color.BgYellow

	a.board = gui.New(cfg)
	a.board.Import(gameBoard)
}

func (a *Application) Board() {
	// if reflect.DeepEqual(a.GB, GameBoard{}) {
	if a.board == nil {
		log.Println("Board empty, downloading the board...")
		b, err := a.downloadBoard()
		if err != nil {
			log.Println(err)
			return
		}
		a.initGameBoard(b.Board)
		log.Println("board downloaded")
	}

	a.board.Display()
}

func (a *Application) Fire(coord string) (string, error) {
	// TODO check if coord is a string [A-J][1-10]
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
	request, err := http.NewRequest("POST", a.Con.Url+fireEndpoint, reader)
	request.Header.Add("X-Auth-Token", a.Con.Token)
	client := &http.Client{}
	res, err := client.Do(request)
	if err != nil {
		log.Println(request, err)
		return "", err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	log.Println("Fire response body", string(body))
	err = json.Unmarshal(body, &fireResponse)
	if err != nil {
		log.Println(err)
		return "", err
	}
	return fireResponse["result"], nil
}

func (a *Application) UpdatePlayerBoard(coords []string) {
	for _, coord := range coords {
		state, err := a.board.HitOrMiss(Left, coord)
		log.Println("Update player board: ", state, coord)
		if err != nil {
			log.Println(err)
		}
		err = a.board.Set(Left, coord, state)
		if err != nil {
			log.Println(err)
		}
	}
	a.board.Display()
}

func (a *Application) UpdateEnemyBoard(coord string) {
	state, err := a.board.HitOrMiss(Right, coord)
	log.Println("Update enemy board: ", state, coord)
	if err != nil {
		log.Println(err)
	}
	err = a.board.Set(Right, coord, state)
	if err != nil {
		log.Println(err)
	}
	a.board.Display()
}
