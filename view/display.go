package view

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	gui "github.com/grupawp/warships-gui/v2"
	"github.com/s22678/statki/connect"
	"github.com/s22678/statki/gamedata"
	"github.com/s22678/statki/lib"
)

type PlayerType int
type TextElem int

// states
const (
	Player PlayerType = iota
	Enemy
)

// textElements
const (
	Turn TextElem = iota
	Timer
	DisplayMessage
)

const (
	shipsCoordsEndpoint = "/api/game/board"
)

var (
	ErrBrokenShips       = errors.New("error downloading ships coordinates")
	ErrPlayerBoardUpdate = errors.New("error while updating player board")
	ErrEnemyBoardUpdate  = errors.New("error while updating enemy board")
)

type WarshipGui struct {
	textElements  []*gui.Text
	boardElements []*gui.Board
	states        [][10][10]gui.State
	ui            *gui.GUI
}

func (wgui *WarshipGui) Play(ctx context.Context, hitMessage chan string, missMessage chan string, timerchan chan string, playerShot chan string) {
	ctx, cancel := context.WithCancel(ctx)
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func(timerchan chan string) {
		defer wg.Done()
		for {
			select {
			case msg := <-timerchan:
				wgui.textElements[Timer].SetText(msg)
			case <-ctx.Done():
				log.Println("terminate timer display")
				return
			}
		}
	}(timerchan)

	wg.Add(1)
	go func(wgr *sync.WaitGroup, hmsg chan string, mmsg chan string, playerShot chan string) {
		defer wgr.Done()
		for {
			select {
			case hitmsg := <-hmsg:
				wgui.textElements[Turn].SetText("Your turn")
				wgui.textElements[DisplayMessage].SetText(hitmsg)
				char := wgui.boardElements[Enemy].Listen(ctx)
				wgui.textElements[DisplayMessage].SetText(fmt.Sprintf("Coordinate: %s", char))
				if char == "" {
					return
				}
				playerShot <- char
				wgui.ui.Log("Coordinate: %s", char) // logs are displayed after the game exits
			case missmsg := <-mmsg:
				wgui.textElements[Turn].SetText("Enemy turn")
				wgui.textElements[DisplayMessage].SetText(missmsg)
				time.Sleep(1 * time.Second)
			case <-ctx.Done():
				log.Println("GAME OVER")
				return
			}
		}
	}(wg, hitMessage, missMessage, playerShot)

	log.Println("starting ui")

	wg.Add(1)
	go func(wgr *sync.WaitGroup) {
		defer wgr.Done()
		wgui.ui.Start(ctx, nil)
		log.Println("terminate GUI display")
		cancel()
	}(wg)

	wg.Wait()
}

func (wgui *WarshipGui) FinishGame(msg string) {
	wgui.textElements[Turn].SetText("GAME OVER")
	wgui.textElements[DisplayMessage].SetText(msg)
}

func CreateGUI(c *connect.Connection) (*WarshipGui, error) {
	wgui := &WarshipGui{}
	wgui.states = append(wgui.states, [10][10]gui.State{})
	wgui.states = append(wgui.states, [10][10]gui.State{})
	log.Println("initiating a new board")
	ui := gui.NewGUI(false)

	// Create player board and add it to []*gui.Board slice as element 0
	wgui.boardElements = append(wgui.boardElements, gui.NewBoard(1, 5, nil))

	coords, err := initPlayerShips(c)
	if err != nil {
		log.Println("(Init)Display:", err)
		return nil, err
	}

	err = wgui.loadPlayerShips(coords)
	if err != nil {
		log.Println("(loadPlayerShips)Display:", err)
		return nil, err
	}
	wgui.boardElements[Player].SetStates(wgui.states[Player])

	// Add player board to GUI
	ui.Draw(wgui.boardElements[Player])

	// Create enemy board and add it to []*gui.Board slice as element 1
	wgui.boardElements = append(wgui.boardElements, gui.NewBoard(50, 5, nil))
	// Add enemy board to GUI
	ui.Draw(wgui.boardElements[Enemy])

	// Create "turn" info and add it to []*gui.Text slice as element 0
	wgui.textElements = append(wgui.textElements, gui.NewText(1, 1, "Press on any coordinate to log it.", nil))
	// Add "turn" info to GUI
	ui.Draw(wgui.textElements[Turn])

	// Create "timer" info and add it to []*gui.Text slice as element 1
	wgui.textElements = append(wgui.textElements, gui.NewText(50, 1, "", nil))
	// Add "timer" info to GUI
	ui.Draw(wgui.textElements[Timer])

	// Create "displayMessage" info and add it to []*gui.Text slice as element 2
	wgui.textElements = append(wgui.textElements, gui.NewText(1, 2, "", nil))
	// Add "displayMessage" info to GUI
	ui.Draw(wgui.textElements[DisplayMessage])

	// Get information about both players from the server
	pi, err := gamedata.GetPlayerInfo(c)
	if err != nil {
		return nil, err
	}

	// Add player nick
	ui.Draw(gui.NewText(1, 28, pi.Nick, nil))

	// Add player description
	for i, line := range word_wrap(pi.Desc) {
		ui.Draw(gui.NewText(1, 30+i, line, nil))
	}

	// Add enemy nick
	ui.Draw(gui.NewText(50, 28, pi.Opponent, nil))

	// Add enemy description
	for i, line := range word_wrap(pi.Opp_desc) {
		ui.Draw(gui.NewText(50, 30+i, line, nil))
	}

	// Add tux
	drawTux(ui)

	wgui.ui = ui

	return wgui, nil
}

func (wgui *WarshipGui) loadPlayerShips(coords []string) error {
	states := [10][10]gui.State{}
	for i := range states {
		states[i] = [10]gui.State{}
	}

	for _, val := range coords {
		setX, setY, err := lib.CoordToIndex(val)
		if err != nil {
			return err
		}
		states[setX][setY] = gui.Ship
	}
	wgui.states[Player] = states
	return nil
}

func (wgui *WarshipGui) UpdatePlayerState(shots []string) error {
	for _, shot := range shots {
		setX, setY, err := lib.CoordToIndex(shot)
		if err != nil {
			return fmt.Errorf("%v: %v", ErrPlayerBoardUpdate, err)
		}
		if wgui.states[Player][setX][setY] == gui.Ship || wgui.states[Player][setX][setY] == gui.Hit {
			wgui.states[Player][setX][setY] = gui.Hit
			continue
		} else {
			wgui.states[Player][setX][setY] = gui.Miss
			continue
		}
	}

	wgui.boardElements[Player].SetStates(wgui.states[Player])
	return nil
}

func (wgui *WarshipGui) UpdateEnemyState(shot, state string) error {
	setX, setY, err := lib.CoordToIndex(shot)
	if err != nil {
		return fmt.Errorf("%v: %v", ErrEnemyBoardUpdate, err)
	}
	if state == "hit" || state == "sunk" {

		wgui.states[Enemy][setX][setY] = gui.Hit
		wgui.boardElements[Enemy].SetStates(wgui.states[Enemy])
		return nil
	}

	if wgui.states[Enemy][setX][setY] == gui.Hit {
		return nil
	}

	wgui.states[Enemy][setX][setY] = gui.Miss
	wgui.boardElements[Enemy].SetStates(wgui.states[Enemy])
	return nil
}

// check neighbouring board elements for possible hits
// func (wgui *WarshipGui) GetNeighbours(xPos, yPos int) []int {

// }

type BoardElemType int

type Corner int
type Border int

type CornerPos Corner
type BorderPos Border

const (
	TopLeft Corner = iota
	BottomLeft
	TopRight
	BottomRight
)

const (
	Top Border = iota
	Bottom
	Right
	Left
)

func getElemType(xPos, yPos int) BoardElemType {

}

// func getBoardElemType(xPos, yPos int) BoardElemType {
// 	if isBorderingType(xPos) && isBorderingType(yPos) {
// 		return Corner
// 	}

// 	if isBorderingType(xPos) || isBorderingType(yPos) {
// 		return Border
// 	}

// 	return Inner
// }

func isBorderingType(pos int) bool {
	if pos == 0 || pos == 9 {
		return true
	}

	return false
}

func isCorner(xPos, yPos int) bool {
	if isBorderingType(xPos) && isBorderingType(yPos) {
		return true
	}

	return false
}

func QuitGame(c *connect.Connection) error {
	_, err := c.GameAPIConnection("DELETE", "/api/game/abandon", nil)
	if err != nil {
		return err
	}
	return nil
}

func initPlayerShips(c *connect.Connection) ([]string, error) {
	if c.Data.Coords == nil || len(c.Data.Coords) == 0 {
		var err error
		playerShipsCoords, err := downloadShips(c)
		if err != nil {
			return nil, fmt.Errorf("board initialization error - cannot download the board: %v", err)
		}
		return playerShipsCoords, nil
	}

	playerShipsCoords := c.Data.Coords

	return playerShipsCoords, nil
}

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

func word_wrap(text string) []string {
	totalLines := len(text)/40 + 2
	sltext := strings.Split(text, " ")
	charCounter := 0
	lineCounter := 0
	lines := make([]string, totalLines)
	spacer := " "
	for _, t := range sltext {
		wordLen := len(t)
		spacer = ""
		if wordLen+charCounter > 40 {
			lineCounter++
			charCounter = 0
		}

		if len(lines[lineCounter]) > 0 {
			spacer = " "
		}

		line := lines[lineCounter] + spacer + t
		lines[lineCounter] = line
		charCounter = len(lines[lineCounter])

	}

	return lines
}

func drawTux(ui *gui.GUI) {
	bodyCfg := &gui.TextConfig{FgColor: gui.White, BgColor: gui.Black}
	yellowCfg := &gui.TextConfig{FgColor: gui.NewColor(255, 216, 1), BgColor: gui.Black}
	ui.Draw(gui.NewText(117, 1, ".88888888:.", bodyCfg))
	ui.Draw(gui.NewText(116, 2, "88888888.88888.", bodyCfg))
	ui.Draw(gui.NewText(114, 3, ".8888888888888888.", bodyCfg))
	ui.Draw(gui.NewText(114, 4, "888888888888888888", bodyCfg))
	ui.Draw(gui.NewText(114, 5, "88' _`88'_  `88888", bodyCfg))
	ui.Draw(gui.NewText(114, 6, "88 88 88 88  88888", bodyCfg))
	ui.Draw(gui.NewText(114, 7, "88_88_", bodyCfg))
	ui.Draw(gui.NewText(120, 7, "::", yellowCfg))
	ui.Draw(gui.NewText(122, 7, "_88_", bodyCfg))
	ui.Draw(gui.NewText(126, 7, ":", yellowCfg))
	ui.Draw(gui.NewText(127, 7, "88888", bodyCfg))

	ui.Draw(gui.NewText(114, 8, "88", bodyCfg))
	ui.Draw(gui.NewText(116, 8, ":::,::,:::::", yellowCfg))
	ui.Draw(gui.NewText(128, 8, "8888", bodyCfg))

	ui.Draw(gui.NewText(114, 9, "88", bodyCfg))
	ui.Draw(gui.NewText(116, 9, "`:::::::::'`", yellowCfg))
	ui.Draw(gui.NewText(128, 9, "8888", bodyCfg))

	ui.Draw(gui.NewText(113, 10, ".88  ", bodyCfg))
	ui.Draw(gui.NewText(118, 10, "`::::'    8:88.", yellowCfg))
	ui.Draw(gui.NewText(118, 10, "`::::'", yellowCfg))
	ui.Draw(gui.NewText(124, 10, "    8:88.", bodyCfg))

	ui.Draw(gui.NewText(112, 11, "8888            `8:888.", bodyCfg))
	ui.Draw(gui.NewText(110, 12, ".8888'             `888888.", bodyCfg))
	ui.Draw(gui.NewText(109, 13, ".8888:..  .::.  ...:'8888888:.", bodyCfg))
	ui.Draw(gui.NewText(108, 14, ".8888.'     :'     `'::`88:88888", bodyCfg))
	ui.Draw(gui.NewText(107, 15, ".8888        '         `.888:8888.", bodyCfg))
	ui.Draw(gui.NewText(106, 16, "888:8         .           888:88888", bodyCfg))
	ui.Draw(gui.NewText(104, 17, ".888:88        .:           888:88888:", bodyCfg))
	ui.Draw(gui.NewText(104, 18, "8888888.       ::           88:888888", bodyCfg))
	ui.Draw(gui.NewText(104, 19, "`", bodyCfg))
	ui.Draw(gui.NewText(105, 19, ".::.", yellowCfg))
	ui.Draw(gui.NewText(109, 19, "888.      ::          .88888888", bodyCfg))

	ui.Draw(gui.NewText(103, 20, ".::::::.", yellowCfg))
	ui.Draw(gui.NewText(111, 20, "888.    ::         ", bodyCfg))
	ui.Draw(gui.NewText(131, 20, ":::", yellowCfg))
	ui.Draw(gui.NewText(134, 20, "`8888'", bodyCfg))
	ui.Draw(gui.NewText(140, 20, ".:.", yellowCfg))

	ui.Draw(gui.NewText(102, 21, "::::::::::.", yellowCfg))
	ui.Draw(gui.NewText(113, 21, "888   '         .", bodyCfg))
	ui.Draw(gui.NewText(130, 21, ":::::::::::::", yellowCfg))

	ui.Draw(gui.NewText(102, 22, "::::::::::::.", yellowCfg))
	ui.Draw(gui.NewText(115, 22, "8    '      .:8", bodyCfg))
	ui.Draw(gui.NewText(130, 22, ":::::::::::::.", yellowCfg))

	ui.Draw(gui.NewText(101, 23, ".::::::::::::::", yellowCfg))
	ui.Draw(gui.NewText(116, 23, ".        .:888", bodyCfg))
	ui.Draw(gui.NewText(130, 23, ":::::::::::::", yellowCfg))

	ui.Draw(gui.NewText(101, 24, ":::::::::::::::", yellowCfg))
	ui.Draw(gui.NewText(116, 24, "88:.__..:88888", bodyCfg))
	ui.Draw(gui.NewText(130, 24, ":::::::::::'", yellowCfg))

	ui.Draw(gui.NewText(102, 25, "`'.:::::::::::", yellowCfg))
	ui.Draw(gui.NewText(116, 25, "88888888888.88", bodyCfg))
	ui.Draw(gui.NewText(130, 25, ":::::::::'", yellowCfg))

	ui.Draw(gui.NewText(108, 26, "`':::_:'", yellowCfg))
	ui.Draw(gui.NewText(116, 26, " -- '' -'-' ", bodyCfg))
	ui.Draw(gui.NewText(128, 26, "`':_::::'", yellowCfg))
}
