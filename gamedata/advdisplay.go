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

// func drawDescription(xPos int, yPos int, text string) {

// 	log.Println("drawDescription", len(text))
// 	re := regexp.MustCompile(`\s+`)
// 	s := re.Split(text, -1)
// 	size := len(s)
// 	length := size / 4
// 	var j, count int
// 	for i := 0; i < size; i += length {
// 		j += length
// 		if j > size {
// 			j = size
// 		}

// 		ui.Draw(advgui.NewText(xPos, yPos+count, strings.Join(s[i:j], " "), nil))
// 		count++
// 	}
// }

func AdvBoard(ctx context.Context, c *connect.Connection, ch chan string, msg chan string, gd *GameStatusData) error {
	log.Println("Creating a board")
	ui = advgui.NewGUI(true)

	// Display player info
	log.Println("Adv board 1")
	ui.Draw(advgui.NewText(1, 28, gd.Nick, nil))
	ui.Draw(advgui.NewText(1, 30, gd.Desc, nil))
	// drawDescription(1, 29, gd.Desc)

	// Display enemy info
	log.Println("Adv board 2")
	ui.Draw(advgui.NewText(50, 28, gd.Opponent, nil))
	ui.Draw(advgui.NewText(50, 32, gd.Opp_desc, nil))
	// drawDescription(50, 29, gd.Opp_desc)

	// Display whose turn is it
	log.Println("Adv board 3")
	turn := advgui.NewText(1, 1, "Press on any coordinate to log it.", nil)
	ui.Draw(turn)

	// Display hit, miss, sunk, win and lose messages
	log.Println("Adv board 4")
	displayMessage := advgui.NewText(1, 2, "", nil)
	ui.Draw(displayMessage)

	// Display the message how to exit the game
	log.Println("Adv board 5")
	ui.Draw(advgui.NewText(1, 3, "Press Ctrl+C to exit", nil))

	// Display player board
	log.Println("Adv board 6")
	playerBoard = advgui.NewBoard(1, 5, nil)

	// Display enemy board
	log.Println("Adv board 7")
	enemyBoard = advgui.NewBoard(50, 5, nil)

	// Draw tux
	log.Println("Adv board 8")
	drawTux(ui)

	// Init player ships on player board
	var err error
	log.Println("Adv board 9")
	InitPlayerShips(c)
	log.Println("Adv board 10")
	PlayerState, err = loadPlayerShips(PlayerShipsCoords)
	if err != nil {
		return err
	}
	log.Println("Adv board 11")
	playerBoard.SetStates(PlayerState)

	log.Println("Adv board 12")
	ui.Draw(playerBoard)
	log.Println("Adv board 13")
	ui.Draw(enemyBoard)

	go func(ctx context.Context, turn *advgui.Text, displayText *advgui.Text, ch chan string, msg chan string) {
		var char string
		log.Println("RUNNING LISTENING CONTEXT FOR THE BOARD")
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
				return
			}
		}
	}(ctx, turn, displayMessage, ch, msg)
	log.Println("starting ui")
	ui.Start(nil)
	return nil
}

func drawTux(ui *advgui.GUI) {
	tuxCfg := &advgui.TextConfig{FgColor: advgui.White, BgColor: advgui.Black}
	ui.Draw(advgui.NewText(100, 5, "                 .88888888:.", tuxCfg))
	ui.Draw(advgui.NewText(100, 6, "                88888888.88888.", tuxCfg))
	ui.Draw(advgui.NewText(100, 7, "              .8888888888888888.", tuxCfg))
	ui.Draw(advgui.NewText(100, 8, "              888888888888888888", tuxCfg))
	ui.Draw(advgui.NewText(100, 9, "              88' _`88'_  `88888", tuxCfg))
	ui.Draw(advgui.NewText(100, 10, "              88 88 88 88  88888", tuxCfg))
	ui.Draw(advgui.NewText(100, 11, "              88_88_::_88_:88888", tuxCfg))
	ui.Draw(advgui.NewText(100, 12, "              88:::,::,:::::8888", tuxCfg))
	ui.Draw(advgui.NewText(100, 13, "              88`:::::::::'`8888", tuxCfg))
	ui.Draw(advgui.NewText(100, 14, "             .88  `::::'    8:88.", tuxCfg))
	ui.Draw(advgui.NewText(100, 15, "            8888            `8:888.", tuxCfg))
	ui.Draw(advgui.NewText(100, 16, "          .8888'             `888888.", tuxCfg))
	ui.Draw(advgui.NewText(100, 17, "         .8888:..  .::.  ...:'8888888:.", tuxCfg))
	ui.Draw(advgui.NewText(100, 18, "        .8888.'     :'     `'::`88:88888", tuxCfg))
	ui.Draw(advgui.NewText(100, 19, "       .8888        '         `.888:8888.", tuxCfg))
	ui.Draw(advgui.NewText(100, 20, "      888:8         .           888:88888", tuxCfg))
	ui.Draw(advgui.NewText(100, 21, "    .888:88        .:           888:88888:", tuxCfg))
	ui.Draw(advgui.NewText(100, 22, "    8888888.       ::           88:888888", tuxCfg))
	ui.Draw(advgui.NewText(100, 23, "    `.::.888.      ::          .88888888", tuxCfg))
	ui.Draw(advgui.NewText(100, 24, "   .::::::.888.    ::         :::`8888'.:.", tuxCfg))
	ui.Draw(advgui.NewText(100, 25, "  ::::::::::.888   '         .::::::::::::", tuxCfg))
	ui.Draw(advgui.NewText(100, 26, "  ::::::::::::.8    '      .:8::::::::::::.", tuxCfg))
	ui.Draw(advgui.NewText(100, 27, " .::::::::::::::.        .:888:::::::::::::", tuxCfg))
	ui.Draw(advgui.NewText(100, 28, " :::::::::::::::88:.__..:88888:::::::::::'", tuxCfg))
	ui.Draw(advgui.NewText(100, 29, "  `'.:::::::::::88888888888.88:::::::::'", tuxCfg))
	ui.Draw(advgui.NewText(100, 30, "        `':::_:' -- '' -'-' `':_::::'", tuxCfg))
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
	// log.Println("SHOTS SHOT BY THE ENEMY", shots)
	for _, shot := range shots {
		setX, setY, err := CoordToIndex(shot)
		if err != nil {
			return fmt.Errorf("error while updating player board %v", err)
		}
		if PlayerState[setX][setY] == advgui.Ship || PlayerState[setX][setY] == advgui.Hit {
			PlayerState[setX][setY] = advgui.Hit
			continue
		} else {
			PlayerState[setX][setY] = advgui.Miss
			continue
		}
	}

	// log.Println("After update:", PlayerState)
	playerBoard.SetStates(PlayerState)
	return nil
}

func UpdateEnemyState(shot, state string) error {
	// log.Println("SHOT SHOT BY THE PLAYER", shot)
	setX, setY, err := CoordToIndex(shot)
	if err != nil {
		return fmt.Errorf("error while updating enemy board %v", err)
	}
	if state == "hit" || state == "sunk" {

		EnemyState[setX][setY] = advgui.Hit
		// log.Printf("ENEMY HIT! shot: %s setx: %d sety: %d\n", shot, setX, setY)
		// log.Println(PlayerState)
	} else {

		if EnemyState[setX][setY] == advgui.Hit {
			EnemyState[setX][setY] = advgui.Hit
		} else {
			EnemyState[setX][setY] = advgui.Miss
		}
		// log.Printf("ENEMY MISSED! shot: %s setx: %d sety: %d\n", shot, setX, setY)
	}

	// log.Println("After update:", EnemyState)
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
