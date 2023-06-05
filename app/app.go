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
	for {
		err = c.InitGame(playWithBot, enemyNick, playerNick, playerDescription, playerShipsCoords)
		if err != nil {
			log.Printf("%v: %v\n", ErrInitGameException, err)
			fmt.Println(ErrInitGameException)
			return
		}
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

	timer := make(chan string)
	statusChan := make(chan *gamedata.Status, 1)
	hitMessage := make(chan string)
	missMessage := make(chan string)
	playerShot := make(chan string)
	wGui, err := view.CreateGUI(c)
	if err != nil {
		log.Println("EEEERRRRRRRRRRORRRRR")
		cancel()
		return
	}

	err = gamedata.LoadHeatMap()
	if err != nil {
		log.Printf("%v: %v\n", ErrLoadHeatmap, err)
	}

	go GetStatusAsync(ctx, statusChan, c)
	go GetTimeAsync(ctx, statusChan, timer, c)

	go func(playerShot chan string, c *connect.Connection, statusChan chan *gamedata.Status, wGui *view.WarshipGui, hitMessage chan string, missMessage chan string) {
		for {
			status := <-statusChan
			if status.Game_status == "game_in_progress" {
				if status.Should_fire {
					hitMessage <- "select coord to shoot"
					wGui.UpdatePlayerState(status.Opp_shots[oppShotsDiff:])
					oppShotsDiff = len(status.Opp_shots)
					for {
						shot := <-playerShot
						resp, err := Fire(c, shot)
						if err != nil {
							log.Printf("Error during firing %v", err)
							break
						}

						err = wGui.UpdateEnemyState(shot, resp)
						if err != nil {
							return
						}

						if resp == "miss" {
							missMessage <- resp
							break
						}

						if resp == "hit" || resp == "sunk" {
							hitMessage <- resp
							status := <-statusChan
							if status.Game_status == "ended" {
								break
							}
							gamedata.AddShotToHeatMap(shot)
							gamedata.SaveHeatMap()
						}
					}
				} else {
					missMessage <- "enemy"
				}

				status = <-statusChan
				time.Sleep(1000 * time.Millisecond)
			}
			if status.Game_status == "ended" {
				log.Println("game status ended - closing the game")
				wGui.UpdatePlayerState(status.Opp_shots[oppShotsDiff:])
				switch status.Last_game_status {
				case "win":
					wGui.FinishGame("game ended: the player is the winner")
				case "lose":
					wGui.FinishGame("game ended: the enemy is the winner")
				case "session not found":
					wGui.FinishGame("game ended: timeout")
				}
				time.Sleep(2 * time.Second)
				cancel()
				return
			}
		}

	}(playerShot, c, statusChan, wGui, hitMessage, missMessage)
	// Print the gameboard
	wGui.Play(ctx, hitMessage, missMessage, timer, playerShot)

	status = <-statusChan
	if status.Game_status == "game_in_progress" {
		log.Println("cancelling and quitting the game")
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
			// log.Println("GET ASYNC TIMER")
			timer <- "--*--"
		}
	}
}

func GetStatusAsync(ctx context.Context, statusChan chan *gamedata.Status, c *connect.Connection) {
	ctx, cancel := context.WithCancel(ctx)
	for {
		select {
		case <-ctx.Done():
			cancel()
			return
		default:
			s, e := gamedata.GetStatus(c)
			if e != nil {
				log.Printf("error getting status: %v\n", e)
				time.Sleep(1 * time.Second)
				continue
			}

			// log.Println("GET ASYNC STATUS")
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
