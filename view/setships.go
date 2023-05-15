package view

import (
	"context"
	"fmt"
	"log"

	gui "github.com/grupawp/warships-gui/v2"
	"github.com/s22678/statki/lib"
)

var (
	coords []string
)

func GetShips() []string {
	return coords
}

func SetShips() error {
	coords = []string{}
	state := [10][10]gui.State{}

	log.Println("Creating a board")
	ui := gui.NewGUI(false)

	// Display whose turn is it
	text := gui.NewText(1, 1, "Press on any coordinate to log it.", nil)
	ui.Draw(text)

	// Display hit, miss, sunk, win and lose messages
	displayMessage := gui.NewText(1, 2, "", nil)
	ui.Draw(displayMessage)

	// Display the message how to exit the game
	ui.Draw(gui.NewText(1, 3, "Press Ctrl+C to exit", nil))

	// Display player board
	board := gui.NewBoard(1, 5, nil)

	ctx, cancel := context.WithCancel(context.Background())

	ui.Draw(board)
	coord := make(chan string)

	go func(coord chan string) {
		for {
			char := board.Listen(ctx)
			if char == "" {
				displayMessage.SetText(fmt.Sprintf("Coordinate: %s", char))
				return
			}
			displayMessage.SetText(fmt.Sprintf("Coordinate: %s", char))
			coord <- char
			ui.Log("Coordinate: %s", char) // logs are displayed after the game exits
		}
	}(coord)
	log.Println("starting ui")

	go func() {
		ui.Start(nil)
		cancel()
	}()

loop:
	for {
		select {
		case t := <-coord:
			setX, setY, err := lib.CoordToIndex(t)
			if err != nil {
				log.Println("error changing coordinates to index when setting ships")
			}
			if state[setX][setY] != gui.Ship {
				coords = append(coords, t)
				state[setX][setY] = gui.Ship
			} else {
				removeCoord(t)
				state[setX][setY] = gui.Empty
			}
			board.SetStates(state)
		case <-ctx.Done():
			break loop
		}
	}

	return nil
}

func removeCoord(coord string) {
	tmp := []string{}
	for _, c := range coords {
		if c != coord {
			tmp = append(tmp, c)
		}
	}
	coords = tmp
}
