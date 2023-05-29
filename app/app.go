package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/s22678/statki/connect"
	"github.com/s22678/statki/gamedata"
	"github.com/s22678/statki/newview"
	"github.com/s22678/statki/view"
)

const (
	Left                  = 0 // indicates player's board (on the left side)
	Right                 = 1 // indicates enemy's board (on the right side)
	maxConnectionAttempts = 10
	boardEndpoint         = "/api/game/board"
	fireEndpoint          = "/api/game/fire"
	descEndpoint          = "/api/game/desc"
	refreshEndpoint       = "/api/game/refresh"
)

var (
	ErrWrongResponseException           = errors.New("there was an error in the response")
	ErrPlayerQuitException              = errors.New("player quit the game")
	ErrWrongCoordinates                 = errors.New("coordinates did not match pattern [A-J]([0-9]|10), try again with correct coordinates")
	ErrGameLoopException                = errors.New("gameloop error when getting status")
	ErrInitGameException                = errors.New("error during game initialization")
	ErrMissingEnemyDescriptionException = errors.New("cannot get enemy name and description")
	ErrGetStatusException               = errors.New("cannot get status")
	ErrLoadHeatmap                      = errors.New("cannot open heatmap file")
	playerDefinedUsername               = false
)

func Fire(c *connect.Connection, coord string) (string, error) {
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
	body, err := c.GameAPIConnection("POST", fireEndpoint, reader)
	if err != nil {
		log.Println("Fire response", err)
		return "", err
	}

	log.Println("Fire response body", string(body))
	err = json.Unmarshal(body, &fireResponse)
	if err != nil {
		log.Println(err)
		return "", err
	}
	return fireResponse["result"], nil
}

func Play(playWithBot bool) {
	c := &connect.Connection{}
	// Initialize the game
	oppShotsDiff := 0
	status := &gamedata.Status{}

	var playerNick string
	var playerDescription string
	// Initialize the game
	if playerDefinedUsername {
		playerNick = c.Data.Nick
		playerDescription = c.Data.Desc
	} else {
		playerNick, _ = GetPlayerInput("set your nickname!")
		playerDescription, _ = GetPlayerInput("set your description!")
		if len(playerNick) > 0 {
			playerDefinedUsername = true
		}
	}
	// playerShipsCoords, _ := GetPlayerInput("set your ships!")
	fmt.Println("Set your ships!")
	time.Sleep(1 * time.Second)
	view.CreateShipyard()
	sh, err := view.CreateShipyard()
	if err != nil {
		log.Println("error creating a shipyard", err)
		return
	}
	sh.SetShips()
	playerShipsCoords := sh.GetShips()
	fmt.Println(playerShipsCoords)
	// playerShipsCoords := strings.Join(gs, " ")
	enemyNick := ""
	if !playWithBot {
		enemyNick, _ = GetPlayerInput("choose your enemy!")
	}
	err = c.InitGame(playWithBot, enemyNick, playerNick, playerDescription, playerShipsCoords)
	if err != nil {
		log.Printf("%v: %v\n", ErrInitGameException, err)
		fmt.Println(ErrInitGameException)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	if playWithBot {

		log.Println("the game has been initiated")

		go func(ctx context.Context) {
			for {
				time.Sleep(10 * time.Second)
				select {
				case <-ctx.Done():
					log.Println("finished refreshing")
					return
				default:
					log.Println("refresh")
					Refresh(c)
				}
			}
		}(ctx)
	}
	// Check if the game has started
	for {
		time.Sleep(1000 * time.Millisecond)
		status, err = gamedata.GetStatus(c)
		if err != nil {
			log.Printf("%v: %v\n", ErrGetStatusException, err)
			fmt.Println(ErrGetStatusException)
			continue
		}
		if status.Game_status == "game_in_progress" {
			log.Println("connection was established")
			fmt.Println("connection was established")
			cancel()
			break
		}
		if status.Game_status == "waiting" {
			log.Println("waiting", status.Game_status)
			fmt.Println("waiting")
		}
	}

	ctx, cancel = context.WithCancel(context.Background())
	shotCoord := make(chan string)
	message := make(chan string)
	timer := make(chan string)
	statusChan := make(chan *gamedata.Status, 1)

	go GetStatusAsync(ctx, statusChan, c)
	go GetTimeAsync(ctx, statusChan, timer, c)

	go func(sh chan string, msg chan string, statusOne *gamedata.Status, c *connect.Connection, statusChan chan *gamedata.Status, timer chan string) {
		err = gamedata.LoadHeatMap()
		if err != nil {
			log.Printf("%v: %v\n", ErrLoadHeatmap, err)
		}
		for {
			status := <-statusChan
			if status.Game_status == "game_in_progress" {
				if status.Should_fire {
					msg <- "player"
					view.UpdatePlayerState(status.Opp_shots[oppShotsDiff:])
					oppShotsDiff = len(status.Opp_shots)
					for {
						shot := <-sh
						resp, err := Fire(c, shot)
						if err != nil {
							log.Printf("Error during firing %v", err)
							break
						}
						err = view.UpdateEnemyState(shot, resp)
						if err != nil {
							return
						}
						msg <- resp
						if resp == "miss" {
							timer <- "--*--"
							break
						}
						if resp == "hit" || resp == "sunk" {
							gamedata.AddShotToHeatMap(shot)
							gamedata.SaveHeatMap()
						}
					}
				} else {
					msg <- "enemy"
				}
				// status, err = gamedata.GetStatus(c)
				status = <-statusChan
				if err != nil {
					log.Println("Error running gameloop", err)
					continue
				}
				time.Sleep(1000 * time.Millisecond)
			}
			if status.Game_status == "ended" {
				view.UpdatePlayerState(status.Opp_shots[oppShotsDiff:])
				switch status.Last_game_status {
				case "win":
					msg <- "game ended: the player is the winner"
				case "lose":
					msg <- "game ended: the enemy is the winner"
				case "session not found":
					msg <- "game ended: timeout"
				}
				time.Sleep(2 * time.Second)
				cancel()
				return
			}
		}

	}(shotCoord, message, status, c, statusChan, timer)
	// Print the gameboard
	err = view.OldGui(ctx, c, shotCoord, message, timer)
	if err != nil {
		log.Println("cannot create a GUI")
		return
	}

	status, err = gamedata.GetStatus(c)
	if err != nil {
		return
	}
	if status.Game_status == "game_in_progress" {
		cancel()
		view.QuitGame(c)
	}
}

func NewPlay(playWithBot bool) {
	c := &connect.Connection{}
	// Initialize the game
	oppShotsDiff := 0
	status := &gamedata.Status{}

	var playerNick string
	var playerDescription string
	// Initialize the game
	if playerDefinedUsername {
		playerNick = c.Data.Nick
		playerDescription = c.Data.Desc
	} else {
		playerNick, _ = GetPlayerInput("set your nickname!")
		playerDescription, _ = GetPlayerInput("set your description!")
		if len(playerNick) > 0 {
			playerDefinedUsername = true
		}
	}
	// playerShipsCoords, _ := GetPlayerInput("set your ships!")
	fmt.Println("Set your ships!")
	time.Sleep(1 * time.Second)
	view.CreateShipyard()
	sh, err := view.CreateShipyard()
	if err != nil {
		log.Println("error creating a shipyard", err)
		return
	}
	sh.SetShips()
	playerShipsCoords := sh.GetShips()
	fmt.Println(playerShipsCoords)
	// playerShipsCoords := strings.Join(gs, " ")
	enemyNick := ""
	if !playWithBot {
		enemyNick, _ = GetPlayerInput("choose your enemy!")
	}
	err = c.InitGame(playWithBot, enemyNick, playerNick, playerDescription, playerShipsCoords)
	if err != nil {
		log.Printf("%v: %v\n", ErrInitGameException, err)
		fmt.Println(ErrInitGameException)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	if playWithBot {

		log.Println("the game has been initiated")

		go func(ctx context.Context) {
			for {
				time.Sleep(10 * time.Second)
				select {
				case <-ctx.Done():
					log.Println("finished refreshing")
					return
				default:
					log.Println("refresh")
					Refresh(c)
				}
			}
		}(ctx)
	}
	// Check if the game has started
	for {
		time.Sleep(1000 * time.Millisecond)
		status, err = gamedata.GetStatus(c)
		if err != nil {
			log.Printf("%v: %v\n", ErrGetStatusException, err)
			fmt.Println(ErrGetStatusException)
			continue
		}
		if status.Game_status == "game_in_progress" {
			log.Println("connection was established")
			fmt.Println("connection was established")
			cancel()
			break
		}
		if status.Game_status == "waiting" {
			log.Println("waiting", status.Game_status)
			fmt.Println("waiting")
		}
	}

	ctx, cancel = context.WithCancel(context.Background())
	shotCoord := make(chan string)

	timer := make(chan string)
	statusChan := make(chan *gamedata.Status, 1)
	hitMessage := make(chan string)
	missMessage := make(chan string)
	wGui, err := newview.NewGui(c)
	if err != nil {
		log.Println("EEEERRRRRRRRRRORRRRR")
		return
	}

	err = gamedata.LoadHeatMap()
	if err != nil {
		log.Printf("%v: %v\n", ErrLoadHeatmap, err)
	}

	go GetStatusAsync(ctx, statusChan, c)
	go GetTimeAsync(ctx, statusChan, timer, c)

	for 

	go func(c *connect.Connection, statusChan chan *gamedata.Status, timer chan string) {

		for {
			status := <-statusChan
			if status.Game_status == "game_in_progress" {
				if status.Should_fire {
					hitMessage <- "select coort to shoot"
					view.UpdatePlayerState(status.Opp_shots[oppShotsDiff:])
					oppShotsDiff = len(status.Opp_shots)
					for {
						shot := <-sh
						resp, err := Fire(c, shot)
						if err != nil {
							log.Printf("Error during firing %v", err)
							break
						}
						err = view.UpdateEnemyState(shot, resp)
						if err != nil {
							return
						}
						msg <- resp
						if resp == "miss" {
							timer <- "--*--"
							break
						}
						if resp == "hit" || resp == "sunk" {
							gamedata.AddShotToHeatMap(shot)
							gamedata.SaveHeatMap()
						}
					}
				} else {
					msg <- "enemy"
				}
				// status, err = gamedata.GetStatus(c)
				status = <-statusChan
				if err != nil {
					log.Println("Error running gameloop", err)
					continue
				}
				time.Sleep(1000 * time.Millisecond)
			}
			if status.Game_status == "ended" {
				view.UpdatePlayerState(status.Opp_shots[oppShotsDiff:])
				switch status.Last_game_status {
				case "win":
					msg <- "game ended: the player is the winner"
				case "lose":
					msg <- "game ended: the enemy is the winner"
				case "session not found":
					msg <- "game ended: timeout"
				}
				time.Sleep(2 * time.Second)
				cancel()
				return
			}
		}

	}(shotCoord, message, status, c, statusChan, timer)
	// Print the gameboard
	err = wg.Play(ctx, c, shotCoord, message, timer)
	if err != nil {
		log.Println("cannot create a GUI")
		return
	}

	status, err = gamedata.GetStatus(c)
	if err != nil {
		return
	}
	if status.Game_status == "game_in_progress" {
		cancel()
		QuitGame(c)
	}
}

func QuitGame(c *connect.Connection) error {
	_, err := c.GameAPIConnection("DELETE", "/api/game/abandon", nil)
	if err != nil {
		return err
	}
	return nil
}

func GetTimeAsync(ctx context.Context, statusChan chan *gamedata.Status, timer chan string, c *connect.Connection) {
	for {
		time.Sleep(1000 * time.Millisecond)
		select {
		case <-ctx.Done():
			return
		case status := <-statusChan:

			if status.Should_fire {
				timer <- fmt.Sprint(status.Timer)
				continue
			}
		}
	}
}

func GetStatusAsync(ctx context.Context, statusChan chan *gamedata.Status, c *connect.Connection) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			s, e := gamedata.GetStatus(c)
			if e != nil {
				log.Printf("error getting timer: %v\n", e)
				time.Sleep(1 * time.Second)
				continue
			}
			statusChan <- s
		}
	}
}

func Refresh(c *connect.Connection) error {
	_, err := c.GameAPIConnection("GET", refreshEndpoint, nil)
	if err != nil {
		return err
	}
	log.Println("Refresh the session:")
	return err
}

func GetPlayerInput(message string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	var input string
	var err error
	fmt.Println(message)

	input, err = reader.ReadString('\n')
	if err != nil {
		log.Println("getPlayerInput:", err)
		fmt.Println("Unexpected error when getting the input, try again")

	}
	input = strings.TrimSpace(input)
	if input == "quit" || input == "exit" {
		return "", ErrPlayerQuitException
	}

	return input, nil
}
