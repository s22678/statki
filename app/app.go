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

func PlayGameAdvGui(playWithBot bool) {
	c := &connect.Connection{}
	// Initialize the game
	oppShotsDiff := 0
	sr := &gamedata.GameStatusData{}

	var playerNick string
	var playerDescription string
	// Initialize the game
	if playerDefinedUsername {
		playerNick, _ = connect.GameConnectionData["nick"].(string)
		playerDescription, _ = connect.GameConnectionData["desc"].(string)
	} else {
		playerNick, _ = GetPlayerInput("set your nickname!")
		playerDescription, _ = GetPlayerInput("set your description!")
		if len(playerNick) > 0 {
			playerDefinedUsername = true
		}
	}
	playerShipsCoords, _ := GetPlayerInput("set your ships!")
	enemyNick := ""
	if !playWithBot {
		enemyNick, _ = GetPlayerInput("choose your enemy!")
	}
	err := c.InitGame(playWithBot, enemyNick, playerNick, playerDescription, playerShipsCoords)
	if err != nil {
		log.Printf("%v: %v", ErrInitGameException, err)
		fmt.Println(ErrInitGameException)
		return
	}

	if playWithBot {

		// Check if the game has started
		for {
			sr, err = gamedata.Status(c)
			if err != nil {
				log.Printf("%v: %v", ErrGetStatusException, err)
				fmt.Println(ErrGetStatusException)
				return
			}
			if sr.Game_status == "game_in_progress" {
				break
			}
			time.Sleep(1000 * time.Millisecond)
		}
	} else {
		log.Println("the game online has been initiated")
		ctx, cancel := context.WithCancel(context.Background())

		go func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					log.Println("finished refreshing")
					return
				default:
					log.Println("refresh")
					Refresh(c)
				}

				time.Sleep(10 * time.Second)
			}
		}(ctx)

		// Check if the game has started
		for {
			sr, err = gamedata.Status(c)
			if err != nil {
				log.Printf("%v: %v", ErrGetStatusException, err)
				fmt.Println(ErrGetStatusException)
				cancel()
				return
			}
			if sr.Game_status == "game_in_progress" {
				log.Println("connection was established")
				fmt.Println("connection was established")
				cancel()
				break
			}
			if sr.Game_status == "waiting" {
				log.Println("waiting", sr.Game_status)
				fmt.Println("waiting")
				time.Sleep(1 * time.Second)
			}
			time.Sleep(1000 * time.Millisecond)
		}
	}

	// Get the enemy name and description
	desc, err := gamedata.Description(c)
	if err != nil {
		log.Printf("%v: %v", ErrMissingEnemyDescriptionException, err)
		fmt.Println(ErrMissingEnemyDescriptionException)
	}
	shotCoord := make(chan string)
	message := make(chan string)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = gamedata.LoadHeatMap()
	if err != nil {
		log.Println("Error loading heatmap", err)
		return
	}
	go func(sh chan string, msg chan string, g *gamedata.GameStatusData, c *connect.Connection) {
		fmt.Println("game started")
		for {
			if g.Game_status == "ended" {
				gamedata.UpdatePlayerState(g.Opp_shots[oppShotsDiff:])
				switch g.Last_game_status {
				case "win":
					msg <- "game ended: the player is the winner"
				case "lose":
					msg <- "game ended: the enemy is the winner"
				case "session not found":
					msg <- "game ended: timeout"
				}
				cancel()
				gamedata.SaveHeatMap()
				return
			}
			if g.Should_fire {
				log.Println("time to fire")
				msg <- "play"
				gamedata.UpdatePlayerState(g.Opp_shots[oppShotsDiff:])
				oppShotsDiff = len(g.Opp_shots)
				for {
					shot := <-sh
					resp, err := Fire(c, shot)
					if err != nil {
						log.Printf("Error during firing %v", err)
						cancel()
						break
					}
					gamedata.UpdateEnemyState(shot, resp)
					msg <- resp
					if resp == "miss" {
						break
					}
					if resp == "hit" || resp == "sunk" {
						gamedata.AddShotToHeatMap(shot)
					}
				}
			} else {
				time.Sleep(1000 * time.Millisecond)
			}
			g, err = gamedata.Status(c)
			if err != nil {
				log.Println("Error running gameloop", err)
				return
			}
		}
	}(shotCoord, message, desc, c)
	err = gamedata.AdvBoard(ctx, c, shotCoord, message, desc)
	if err != nil {
		log.Println("cannot create an advanced GUI")
		return
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
