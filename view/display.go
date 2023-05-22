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
type GuiElem int
type TextElem int

const (
	Player PlayerType = iota
	Enemy
)

const (
	PlayerBoard GuiElem = iota
	EnemyBoard
)

const (
	Turn TextElem = iota
	Timer
	DisplayMessage
)

const (
	shipsCoordsEndpoint = "/api/game/board"
)

var (
	playerBoard          = &gui.Board{}
	enemyBoard           = &gui.Board{}
	PlayerState          = [10][10]gui.State{}
	EnemyState           = [10][10]gui.State{}
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

func (wg *WarshipGui) Play() {
	ctx, cancel := context.WithCancel(context.Background())
	wgr := &sync.WaitGroup{}

	wgr.Add(1)
	go func(wgr *sync.WaitGroup) {
		defer wgr.Done()
		for {
			char := wg.boardElements[Enemy].Listen(ctx)
			wg.textElements[DisplayMessage].SetText(fmt.Sprintf("Coordinate: %s", char))
			if char == "" {
				return
			}
			wg.ui.Log("Coordinate: %s", char) // logs are displayed after the game exits
		}
	}(wgr)
	log.Println("starting ui")

	wgr.Add(1)
	go func(wgr *sync.WaitGroup) {
		defer wgr.Done()
		wg.ui.Start(context.TODO(), nil)
		cancel()
	}(wgr)

	wgr.Wait()
}

func NewGui(c *connect.Connection) (*WarshipGui, error) {
	wg := &WarshipGui{}
	wg.states = append(wg.states, [10][10]gui.State{})
	wg.states = append(wg.states, [10][10]gui.State{})
	log.Println("initiating a new board")
	ui := gui.NewGUI(false)

	// Create player board and add it to []*gui.Board slice as element 0
	wg.boardElements = append(wg.boardElements, gui.NewBoard(1, 5, nil))

	coords, err := initPlayerShips(c)
	if err != nil {
		log.Println("(Init)Display:", err)
		return nil, err
	}

	err = wg.loadPlayerShips(coords)
	if err != nil {
		log.Println("(loadPlayerShips)Display:", err)
		return nil, err
	}
	wg.boardElements[Player].SetStates(wg.states[Player])

	// Add player board to GUI
	ui.Draw(wg.boardElements[Player])

	// Create enemy board and add it to []*gui.Board slice as element 1
	wg.boardElements = append(wg.boardElements, gui.NewBoard(50, 5, nil))
	// Add enemy board to GUI
	ui.Draw(wg.boardElements[Enemy])

	// Create "turn" info and add it to []*gui.Text slice as element 0
	wg.textElements = append(wg.textElements, gui.NewText(1, 1, "Press on any coordinate to log it.", nil))
	// Add "turn" info to GUI
	ui.Draw(wg.textElements[Turn])

	// Create "timer" info and add it to []*gui.Text slice as element 1
	wg.textElements = append(wg.textElements, gui.NewText(50, 1, "", nil))
	// Add "timer" info to GUI
	ui.Draw(wg.textElements[Timer])

	// Create "displayMessage" info and add it to []*gui.Text slice as element 2
	wg.textElements = append(wg.textElements, gui.NewText(1, 2, "", nil))
	// Add "displayMessage" info to GUI
	ui.Draw(wg.textElements[DisplayMessage])

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

	wg.ui = ui

	return wg, nil
}

func (wg *WarshipGui) loadPlayerShips(coords []string) error {
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
	wg.states[Player] = states
	return nil
}

func (wg *WarshipGui) UpdatePlayerState(shots []string) error {
	for _, shot := range shots {
		setX, setY, err := lib.CoordToIndex(shot)
		if err != nil {
			return fmt.Errorf("%v: %v", ErrPlayerBoardUpdate, err)
		}
		if wg.states[Player][setX][setY] == gui.Ship || wg.states[Player][setX][setY] == gui.Hit {
			wg.states[Player][setX][setY] = gui.Hit
			continue
		} else {
			wg.states[Player][setX][setY] = gui.Miss
			continue
		}
	}

	playerBoard.SetStates(PlayerState)
	return nil
}

func (wg *WarshipGui) UpdateEnemyState(shot, state string) error {
	setX, setY, err := lib.CoordToIndex(shot)
	if err != nil {
		return fmt.Errorf("%v: %v", ErrEnemyBoardUpdate, err)
	}
	if state == "hit" || state == "sunk" {

		wg.states[Enemy][setX][setY] = gui.Hit
		enemyBoard.SetStates(wg.states[Enemy])
		return nil
	}

	if wg.states[Enemy][setX][setY] == gui.Hit {
		return nil
	}

	wg.states[Enemy][setX][setY] = gui.Miss
	enemyBoard.SetStates(wg.states[Enemy])
	return nil
}

func OldGui(ctx context.Context, c *connect.Connection, ch chan string, msg chan string, timerchan chan string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	pi, err := gamedata.GetPlayerInfo(c)
	if err != nil {
		return err
	}
	log.Println("Creating a board")
	ui := gui.NewGUI(true)

	// Display player info
	ui.Draw(gui.NewText(1, 28, pi.Nick, nil))
	playerDescritpion := word_wrap(pi.Desc)
	for i, line := range playerDescritpion {
		ui.Draw(gui.NewText(1, 30+i, line, nil))
	}

	// Display enemy info
	ui.Draw(gui.NewText(50, 28, pi.Opponent, nil))
	enemyDescription := word_wrap(pi.Opp_desc)
	for i, line := range enemyDescription {
		ui.Draw(gui.NewText(50, 30+i, line, nil))
	}

	// Display whose turn is it
	turn := gui.NewText(1, 1, "Press on any coordinate to log it.", nil)
	ui.Draw(turn)

	// Display time left until the end of the turn
	timer := gui.NewText(50, 1, "", nil)
	ui.Draw(timer)

	// Display hit, miss, sunk, win and lose messages
	displayMessage := gui.NewText(1, 2, "", nil)
	ui.Draw(displayMessage)

	// Display the message how to exit the game
	ui.Draw(gui.NewText(1, 3, "Press Ctrl+C to exit", nil))

	// Display player board
	playerBoard = gui.NewBoard(1, 5, nil)

	// Display enemy board
	enemyBoard = gui.NewBoard(50, 5, nil)

	// Draw tux
	drawTux(ui)

	// Init player ships on player board
	coords, err := initPlayerShips(c)
	if err != nil {
		log.Println("(Init)Display:", err)
		return err
	}
	PlayerState, err = loadPlayerShips(coords)
	if err != nil {
		log.Println("(loadPlayerShips)Display:", err)
		return err
	}
	playerBoard.SetStates(PlayerState)

	ui.Draw(playerBoard)
	ui.Draw(enemyBoard)

	go func(timerchan chan string) {
		for {
			select {
			case msg := <-timerchan:
				timer.SetText(msg)
			case <-ctx.Done():
				return
			}
		}
	}(timerchan)

	go func(turn *gui.Text, displayMessage *gui.Text, ch chan string, msg chan string) {
		var char string
		for {
			select {
			case cmd := <-msg:
				switch cmd {
				case "hit", "sunk", "player":
					turn.SetText("Your turn")
					if cmd == "hit" {
						displayMessage.SetText("hit!!")
					}
					if cmd == "sunk" {
						displayMessage.SetText("you've sunk enemy ship! congratulations!!")
					}
					if cmd == "player" {
						turn.SetText("Your turn")
					}
					char = enemyBoard.Listen(ctx)
					displayMessage.SetText(fmt.Sprintf("Coordinate: %s", char))
					ch <- char
					ui.Log("Coordinate: %s", char) // logs are displayed after the game exits
				case "miss", "enemy":
					turn.SetText("Enemy turn")
					time.Sleep(1 * time.Second)
				default:
					displayMessage.SetText(cmd)
				}

			case <-ctx.Done():
				log.Println("GAME OVER", pi.Nick)
				return
			}
		}
	}(turn, displayMessage, ch, msg)
	log.Println("starting ui")
	ui.Start(ctx, nil)
	cancel()
	return nil
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

func UpdatePlayerState(shots []string) error {
	for _, shot := range shots {
		setX, setY, err := lib.CoordToIndex(shot)
		if err != nil {
			return fmt.Errorf("%v %v", ErrPlayerBoardUpdate, err)
		}
		if PlayerState[setX][setY] == gui.Ship || PlayerState[setX][setY] == gui.Hit {
			PlayerState[setX][setY] = gui.Hit
			continue
		} else {
			PlayerState[setX][setY] = gui.Miss
			continue
		}
	}

	playerBoard.SetStates(PlayerState)
	return nil
}

func UpdateEnemyState(shot, state string) error {
	setX, setY, err := lib.CoordToIndex(shot)
	if err != nil {
		return fmt.Errorf("%v %v", ErrEnemyBoardUpdate, err)
	}
	if state == "hit" || state == "sunk" {

		EnemyState[setX][setY] = gui.Hit
	} else {

		if EnemyState[setX][setY] == gui.Hit {
			EnemyState[setX][setY] = gui.Hit
		} else {
			EnemyState[setX][setY] = gui.Miss
		}
	}
	enemyBoard.SetStates(EnemyState)
	return nil
}

func loadPlayerShips(coords []string) ([10][10]gui.State, error) {
	states := [10][10]gui.State{}
	for i := range states {
		states[i] = [10]gui.State{}
	}

	for _, val := range coords {
		setX, setY, err := lib.CoordToIndex(val)
		if err != nil {
			return states, err
		}
		states[setX][setY] = gui.Ship
	}
	return states, nil
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
