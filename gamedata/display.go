package gamedata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	gui "github.com/grupawp/warships-gui/v2"
	"github.com/s22678/statki/connect"
	"github.com/takuoki/clmconv"
)

const (
	shipsCoordsEndpoint = "/api/game/board"
)

var (
	playerBoard          = &gui.Board{}
	enemyBoard           = &gui.Board{}
	PlayerShipsCoords    = []string{}
	PlayerState          = [10][10]gui.State{}
	EnemyState           = [10][10]gui.State{}
	ErrBrokenShips       = errors.New("error downloading ships coordinates")
	ErrPlayerBoardUpdate = errors.New("error while updating player board")
	ErrEnemyBoardUpdate  = errors.New("error while updating enemy board")
)

func AdvBoard(ctx context.Context, c *connect.Connection, ch chan string, msg chan string, gd *GameStatusData) error {
	log.Println("Creating a board")
	ui := gui.NewGUI(true)

	// Display player info
	ui.Draw(gui.NewText(1, 28, gd.Nick, nil))
	playerDescritpion := word_wrap(gd.Desc)
	// ui.Draw(gui.NewText(1, 30, gd.Desc, nil))
	for i, line := range playerDescritpion {
		ui.Draw(gui.NewText(1, 30+i, strings.Join(line, " "), nil))
	}

	// Display enemy info
	ui.Draw(gui.NewText(50, 28, gd.Opponent, nil))
	enemyDescription := word_wrap(gd.Opp_desc)
	for i, line := range enemyDescription {
		ui.Draw(gui.NewText(50, 30+i, strings.Join(line, " "), nil))
	}
	// ui.Draw(gui.NewText(50, 32, gd.Opp_desc, nil))
	// ui.Draw(gui.NewText(50, 32, enemyDescription, nil))

	// Display whose turn is it
	turn := gui.NewText(1, 1, "Press on any coordinate to log it.", nil)
	ui.Draw(turn)

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
	var err error
	InitPlayerShips(c)
	PlayerState, err = loadPlayerShips(PlayerShipsCoords)
	if err != nil {
		return err
	}
	playerBoard.SetStates(PlayerState)

	ui.Draw(playerBoard)
	ui.Draw(enemyBoard)

	go func(ctx context.Context, turn *gui.Text, displayText *gui.Text, ch chan string, msg chan string) {
		var char string
		for {
			select {
			case cmd := <-msg:
				switch cmd {
				case "hit", "sunk", "play":
					turn.SetText("Your turn")
					if cmd == "hit" {
						displayText.SetText("hit!!")
					}
					if cmd == "sunk" {
						displayText.SetText("you've sunk enemy ship! congratulations!!")
					}
					char = enemyBoard.Listen(context.TODO())
					displayText.SetText(fmt.Sprintf("Coordinate: %s", char))
					ch <- char
					ui.Log("Coordinate: %s", char) // logs are displayed after the game exits
				case "miss":
					turn.SetText("Enemy turn")
					time.Sleep(1 * time.Second)
				default:
					displayText.SetText(cmd)
				}
			case <-ctx.Done():
				log.Println("GAME OVER", gd.Nick)
				QuitGame(c)
				return
			}
		}
	}(ctx, turn, displayMessage, ch, msg)
	log.Println("starting ui")
	ui.Start(nil)

	return nil
}

func QuitGame(c *connect.Connection) error {
	_, err := c.GameAPIConnection("DELETE", "/api/game/abandon", nil)
	if err != nil {
		return err
	}
	return nil
}

// 43

func InitPlayerShips(c *connect.Connection) error {
	if connect.GameConnectionData["coords"] == nil {
		var err error
		PlayerShipsCoords, err = DownloadShips(c)
		if err != nil {
			return fmt.Errorf("board initialization error - cannot download the board: %v", err)
		}
	} else {
		var ok bool
		PlayerShipsCoords, ok = connect.GameConnectionData["coords"].([]string)
		if !ok {
			log.Println("board initialization error - wrong type assertion")
			return fmt.Errorf("board initialization error - wrong type assertion")
		}
	}
	return nil
}

func DownloadShips(c *connect.Connection) ([]string, error) {
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
		setX, setY, err := CoordToIndex(shot)
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
	setX, setY, err := CoordToIndex(shot)
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
		setX, setY, err := CoordToIndex(val)
		if err != nil {
			return states, err
		}
		states[setX][setY] = gui.Ship
	}
	return states, nil
}

// Modified coordinates like A1, B2 to (int, int) pair. A1 = (1,1), B3 = (2,3) etc.
func CoordToIndex(coord string) (setx int, sety int, err error) {
	setx, err = clmconv.Atoi(coord[:1])
	if err != nil {
		return 0, 0, err
	}
	if len(coord) == 2 {
		sety, err = strconv.Atoi(coord[1:2])
		if err != nil {
			return 0, 0, err
		}
	}
	if len(coord) >= 3 {
		sety, err = strconv.Atoi(coord[1:3])
		if err != nil {
			return 0, 0, err
		}
	}
	return setx, sety - 1, nil
}

func word_wrap(text string) [][]string {

	maxLength := 40
	length := len(text)
	if length == 0 {
		return [][]string{}
	}

	offset := 0
	mod := length % maxLength
	if mod != 0 {
		offset = 1
	}

	lines := length/maxLength + offset
	whole := make([][]string, length)
	i := 1
	for {
		if i == lines {
			break
		}
		whole[i-1] = []string{text[maxLength*(i-1) : maxLength*i]}
		i++
	}
	whole[i-1] = []string{text[maxLength*(i-1):]}
	return whole
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
