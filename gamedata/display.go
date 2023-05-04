package gamedata

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/fatih/color"
	gui "github.com/grupawp/warships-lightgui/v2"
	"github.com/s22678/statki/connect"
)

const (
	Left = iota
	Right
	shipsCoordsEndpoint = "/api/game/board"
)

var (
	ErrBrokenShips = errors.New("error downloading ships coordinates")
	board          = &gui.Board{}
)

func downloadShips(c *connect.Connection) ([]string, error) {
	coords := make(map[string][]string)
	body, err := c.GameAPIConnection("GET", shipsCoordsEndpoint, nil)
	if err != nil {
		err = fmt.Errorf("%w %w", err, ErrBrokenShips)
		log.Println(err)
		return nil, err
	}

	err = json.Unmarshal(body, &coords)
	if err != nil {
		err = fmt.Errorf("%w %w", err, ErrBrokenShips)
		log.Println(err)
		return nil, err
	}
	log.Println("ships positions:", coords["board"])
	return coords["board"], nil
}

func initGameBoard() {
	cfg := gui.NewConfig()
	cfg.HitChar = '#'
	cfg.HitColor = color.FgRed
	cfg.BorderColor = color.BgRed
	cfg.RulerTextColor = color.BgYellow

	board = gui.New(cfg)
}

func Board(c *connect.Connection) error {
	log.Println("Creating a new board...")
	initGameBoard()
	var ships []string
	var ok bool
	if connect.GameConnectionData["coords"] == nil {
		var err error
		log.Println("downloading ships...")
		ships, err = downloadShips(c)
		if err != nil {
			return fmt.Errorf("GetBoard error: %w", err)
		}
		log.Println("ships downloaded")
	} else {
		// Type assertion
		ships, ok = connect.GameConnectionData["coords"].([]string)
		if !ok {
			log.Println("board initialization error - wrong type assertion")
			return fmt.Errorf("board initialization error - wrong type assertion")
		}
	}

	fmt.Println("Adding ships to the board...")
	board.Import(ships)
	log.Println("Ships added")

	board.Display()
	return nil
}

func UpdatePlayerBoard(coords []string) {
	log.Println("Update player board: ", coords)
	for _, coord := range coords {
		state, err := board.HitOrMiss(Left, coord)
		if err != nil {
			log.Println(err)
		}
		err = board.Set(Left, coord, state)
		if err != nil {
			log.Println(err)
		}
	}
	board.Display()
}

func UpdateEnemyBoard(coord string) {
	state, err := board.HitOrMiss(Right, coord)
	log.Println("Update enemy board: ", state, coord)
	if err != nil {
		log.Println(err)
	}
	err = board.Set(Right, coord, state)
	if err != nil {
		log.Println(err)
	}
	board.Display()
}
