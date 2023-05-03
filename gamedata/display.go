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
	body, err := c.GameAPIConnection("GET", shipsCoordsEndpoint)
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

func GetBoard(c *connect.Connection) {
	log.Println("Creating a new board...")
	initGameBoard()
	log.Println("downloading ships...")
	ships, err := downloadShips(c)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("Adding ships to the board...")
	board.Import(ships)
	log.Println("board downloaded")

	board.Display()
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
