package gamedata

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	advgui "github.com/grupawp/warships-gui/v2"
	"github.com/s22678/statki/connect"
	"github.com/takuoki/clmconv"
)

var (
	playerBoard       = &advgui.Board{}
	enemyBoard        = &advgui.Board{}
	ui                = &advgui.GUI{}
	PlayerShipsCoords = []string{}
	PlayerState       = [10][10]advgui.State{}
	EnemyState        = [10][10]advgui.State{}
)

func AdvBoard(c *connect.Connection, ch chan string, msg chan string, gd *GameStatusData) error {

	ui = advgui.NewGUI(true)

	ui.Draw(advgui.NewText(1, 40, gd.Nick+" "+gd.Desc, nil))
	ui.Draw(advgui.NewText(50, 41, gd.Opponent+" "+gd.Opp_desc, nil))

	txt := advgui.NewText(1, 1, "Press on any coordinate to log it.", nil)
	ui.Draw(txt)
	displayMessage := advgui.NewText(1, 2, "", nil)
	ui.Draw(displayMessage)
	ui.Draw(txt)
	ui.Draw(advgui.NewText(1, 3, "Press Ctrl+C to exit", nil))
	playerBoard = advgui.NewBoard(1, 5, nil)
	enemyBoard = advgui.NewBoard(50, 5, nil)

	var err error
	InitPlayerShips(c)
	PlayerState, err = loadPlayerShips(PlayerShipsCoords)
	if err != nil {
		return err
	}
	playerBoard.SetStates(PlayerState)

	ui.Draw(playerBoard)
	ui.Draw(enemyBoard)

	go func(text *advgui.Text, displayText *advgui.Text, ch chan string, msg chan string) {
		var char string
		for {
			select {
			case cmd := <-msg:
				switch cmd {
				case "play":
					displayText.SetText("Your turn")
					char = enemyBoard.Listen(context.TODO())
					text.SetText(fmt.Sprintf("Coordinate: %s", char))
					ch <- char
					ui.Log("Coordinate: %s", char) // logs are displayed after the game exits
				case "pause":
					displayText.SetText("Enemy turn")
					time.Sleep(1 * time.Second)
				default:
					displayText.SetText(cmd)
				}
			}
		}
	}(txt, displayMessage, ch, msg)
	ui.Start(nil)
	return nil
}

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

func UpdatePlayerState(shots []string) error {
	log.Println("SHOTS SHOT BY THE ENEMY", shots)
	for _, shot := range shots {
		setX, setY, err := CoordToIndex(shot)
		if err != nil {
			return fmt.Errorf("error while updating player board %v", err)
		}
		if PlayerState[setX][setY] == advgui.Ship || PlayerState[setX][setY] == advgui.Hit {
			PlayerState[setX][setY] = advgui.Hit
			log.Printf("PLAYER HIT! shot: %s setx: %d sety: %d\n", shot, setX, setY)
			log.Println(PlayerState)
			continue
		} else {
			PlayerState[setX][setY] = advgui.Miss
			log.Printf("PLAYER MISSED! shot: %s setx: %d sety: %d\n", shot, setX, setY)
			continue
		}
	}

	log.Println("After update:", PlayerState)
	playerBoard.SetStates(PlayerState)
	return nil
}

func UpdateEnemyState(shot, state string) error {
	log.Println("SHOT SHOT BY THE PLAYER", shot)
	setX, setY, err := CoordToIndex(shot)
	if err != nil {
		return fmt.Errorf("error while updating enemy board %v", err)
	}
	if state == "hit" || state == "sunk" {

		EnemyState[setX][setY] = advgui.Hit
		log.Printf("ENEMY HIT! shot: %s setx: %d sety: %d\n", shot, setX, setY)
		log.Println(PlayerState)
	} else {

		EnemyState[setX][setY] = advgui.Miss
		log.Printf("ENEMY MISSED! shot: %s setx: %d sety: %d\n", shot, setX, setY)
	}

	log.Println("After update:", EnemyState)
	enemyBoard.SetStates(EnemyState)
	return nil
}

func loadPlayerShips(coords []string) ([10][10]advgui.State, error) {
	states := [10][10]advgui.State{}
	for i := range states {
		states[i] = [10]advgui.State{}
	}

	for _, val := range coords {
		setX, setY, err := CoordToIndex(val)
		if err != nil {
			return states, err
		}
		states[setX][setY] = advgui.Ship
	}
	return states, nil
}

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
