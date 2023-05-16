package view

import (
	"context"
	"fmt"
	"log"
	"time"

	gui "github.com/grupawp/warships-gui/v2"
	"github.com/s22678/statki/lib"
)

type Shipyard struct {
	displayMessage *gui.Text
	board          *gui.Board
	coords         []string
	state          [10][10]gui.State
	ui             *gui.GUI
}

func (s *Shipyard) GetShips() []string {
	return s.coords
}

func CreateShipyard() (*Shipyard, error) {
	s := &Shipyard{}
	s.coords = []string{}
	s.state = [10][10]gui.State{}

	log.Println("Creating a board")
	s.ui = gui.NewGUI(false)

	// Display hit, miss, sunk, win and lose messages
	s.displayMessage = gui.NewText(1, 2, "", nil)
	s.ui.Draw(s.displayMessage)

	// Display the message how to exit the game
	s.ui.Draw(gui.NewText(1, 3, "Press Ctrl+C to exit", nil))

	// Display player board
	s.board = gui.NewBoard(1, 5, nil)
	s.ui.Draw(s.board)
	return s, nil
}

func (s *Shipyard) SetShips() error {

	ctx, cancel := context.WithCancel(context.Background())
	coord := make(chan string)

	go func(coord chan string) {
		for {
			char := s.board.Listen(ctx)
			if char == "" {
				s.displayMessage.SetText(fmt.Sprintf("Coordinate: %s", char))
				return
			}
			// s.displayMessage.SetText(fmt.Sprintf("Coordinate: %s", char))
			coord <- char
			s.ui.Log("Coordinate: %s", char) // logs are displayed after the game exits
		}
	}(coord)
	log.Println("starting ui")

	go func() {
		s.ui.Start(ctx, nil)
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
			if s.state[setX][setY] != gui.Ship {
				s.coords = append(s.coords, t)
				s.state[setX][setY] = gui.Ship
			} else {
				s.removeCoord(t)
				s.state[setX][setY] = gui.Empty
			}
			s.displayMessage.SetText(fmt.Sprintf("Coordinates: %v", s.coords))
			s.board.SetStates(s.state)
			if len(s.coords) >= 20 {
				time.Sleep(1500 * time.Millisecond)
				cancel()
			}
		case <-ctx.Done():
			break loop
		}
	}

	return nil
}

func (s *Shipyard) removeCoord(coord string) {
	tmp := []string{}
	for _, c := range s.coords {
		if c != coord {
			tmp = append(tmp, c)
		}
	}
	s.coords = tmp
}

type point struct {
	x, y int
}

func (s *Shipyard) drawBorder(p point) {
	//
	//    XXX
	//    XOX
	//    XXX
	//
	vec := []point{
		{-1, 0},
		{-1, -1},
		{0, 1},
		{1, 1},
		{1, 0},
		{-1, 1},
		{0, -1},
		{1, -1},
	}

	for _, v := range vec {
		dx := p.x + v.x
		dy := p.y + v.y

		if dx < 0 || dx >= 10 || dy < 0 || dy >= 10 {
			continue
		}

		prev := s.state[dx][dy]
		if !(prev == gui.Ship || prev == gui.Hit || prev == gui.Miss) { // don't overwrite already marked
			s.state[dx][dy] = gui.Ship
		}
	}
}

func (s *Shipyard) searchElement(x, y int, points *[]point) {
	vec := []point{
		{-1, 0},
		{0, 1},
		{1, 0},
		{0, -1},
	}

	for _, i := range *points {
		if i.x == x && i.y == y {
			return
		}
	}

	*points = append(*points, point{x, y})
	connections := []point{}

	for _, v := range vec {
		dx := x + v.x
		dy := y + v.y

		if dx < 0 || dx >= 10 || dy < 0 || dy >= 10 {
			continue
		}

		if s.state[x+v.x][y+v.y] == gui.Ship || s.state[x+v.x][y+v.y] == gui.Hit {
			connections = append(connections, point{dx, dy})
		}
	}

	// Run the method recursively on each linked element
	for _, c := range connections {
		s.searchElement(c.x, c.y, points)
	}
}

// CreateBorder creates a border around a (sunken) ship, to indicate
// which coordinates cannot contain a ship segment and can be safely
// ignored.
func (s *Shipyard) CreateBorder(coord string) {
	x, y, err := lib.CoordToIndex(coord)
	if err != nil {
		log.Println(err)
		return
	}

	points := []point{}
	s.searchElement(x, y, &points)

	for _, i := range points {
		s.drawBorder(i)
	}
}
