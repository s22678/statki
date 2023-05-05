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
	"regexp"
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

func QuitGame(c *connect.Connection) error {
	_, err := c.GameAPIConnection("DELETE", "/api/game/abandon", nil)
	if err != nil {
		return err
	}
	return nil
}

func PlayGameAdvGui(c *connect.Connection, playWithBot bool) {
	// Initialize the game
	oppShotsDiff := 0
	sr := &gamedata.GameStatusData{}

	// Initialize the game
	playerNick, _ := GetPlayerInput("set your nickname!", false)
	playerDescription, _ := GetPlayerInput("set your description!", false)
	playerShipsCoords, _ := GetPlayerInput("set your ships!", false)
	enemyNick, _ := GetPlayerInput("choose your enemy!", false)
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
			time.Sleep(500 * time.Millisecond)
		}
	} else {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					return
				default:
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
				break
			}
			if sr.Game_status == "waiting" {
				fmt.Println(sr.Game_status)
				time.Sleep(1 * time.Second)
			}
			time.Sleep(500 * time.Millisecond)
		}
		cancel()
	}

	// Get the enemy name and description
	desc, err := gamedata.Description(c)
	if err != nil {
		log.Printf("%v: %v", ErrMissingEnemyDescriptionException, err)
		fmt.Println(ErrMissingEnemyDescriptionException)
	}
	shotCoord := make(chan string)
	message := make(chan string)

	go func(sh chan string, msg chan string, g *gamedata.GameStatusData, c *connect.Connection) {
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
				return
			}
			if g.Should_fire {
				msg <- "play"
				gamedata.UpdatePlayerState(g.Opp_shots[oppShotsDiff:])
				oppShotsDiff = len(g.Opp_shots)
				for {
					shot := <-sh
					resp, err := Fire(c, shot)
					if err != nil {
						log.Fatalf("Error during firing %v", err)
					}
					gamedata.UpdateEnemyState(shot, resp)
					msg <- resp
					if resp == "miss" {
						msg <- "pause"
						break
					} else {
						msg <- "play"
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
	}(shotCoord, message, sr, c)
	err = gamedata.AdvBoard(c, shotCoord, message, desc)
	if err != nil {
		log.Println("cannot create an advanced GUI")
		return
	}
}

func PlayGame(c *connect.Connection, playWithBot bool) {
	oppShotsDiff := 0
	turnCounter := 1
	fireResponse := ""
	sr := &gamedata.GameStatusData{}

	// Initialize the game
	playerNick, _ := GetPlayerInput("set your nickname!", false)
	playerDescription, _ := GetPlayerInput("set your description!", false)
	playerShipsCoords, _ := GetPlayerInput("set your ships!", false)
	enemyNick, _ := GetPlayerInput("choose your enemy!", false)
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
			time.Sleep(500 * time.Millisecond)
		}
	} else {
		ctx, cancel := context.WithCancel(context.Background())

		go func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					return
				default:
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
				break
			}
			if sr.Game_status == "waiting" {
				fmt.Println(sr.Game_status)
				time.Sleep(1 * time.Second)
			}
			time.Sleep(500 * time.Millisecond)
		}
		cancel()
	}

	// Get the enemy name and description
	desc, err := gamedata.Description(c)
	if err != nil {
		log.Printf("%v: %v", ErrMissingEnemyDescriptionException, err)
		fmt.Println(ErrMissingEnemyDescriptionException)
	}

	// Display ships and enemy description
	err = gamedata.Board(c)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("You're playing against: ", desc.Opponent, desc.Opp_desc)

	// Run until the game has ended
gameloop:
	for {
		if sr.Game_status == "ended" {
			gamedata.UpdatePlayerBoard(sr.Opp_shots[oppShotsDiff:])
			fmt.Println("You're playing against: ", desc.Opponent, desc.Opp_desc)
			switch sr.Last_game_status {
			case "win":
				fmt.Println("game ended: the player is the winner")
				log.Println("game ended: the player is the winner")
			case "lose":
				fmt.Println("game ended: the enemy is the winner")
				log.Println("game ended: the enemy is the winner")
			case "session not found":
				fmt.Println("game ended: timeout")
				log.Println("game ended: timeout")
			}
			time.Sleep(3 * time.Second)
			break gameloop
		}

		if sr.Should_fire {
			log.Println("player turn:", turnCounter)
			// log.Printf("DEBUG: %d %s", oppShotsDiff, sr.Opp_shots[oppShotsDiff:])
			gamedata.UpdatePlayerBoard(sr.Opp_shots[oppShotsDiff:])
			fmt.Println("You're playing against: ", desc.Opponent, desc.Opp_desc)
			for {
				fireResponse, err = fire(c, desc)
				if fireResponse == "hit" || fireResponse == "sunk" {
					log.Println("DEBUG: inside fire loop")
				} else if err == ErrPlayerQuitException || err == connect.ErrSessionNotFoundException {
					log.Println(err)
					QuitGame(c)
					break gameloop
				} else {
					break
				}
			}
			log.Println("enemy turn", turnCounter)
			fmt.Println("enemy turn", turnCounter)
			turnCounter++
			oppShotsDiff = len(sr.Opp_shots)
		} else {
			time.Sleep(1000 * time.Millisecond)
		}
		sr, err = gamedata.Status(c)
		if err != nil {
			log.Println("can't get the status, exiting", err)
			fmt.Println("can't get the status, exiting", err)
			return
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

func fire(c *connect.Connection, desc *gamedata.GameStatusData) (string, error) {
	input, err := GetPlayerInput("your move!", true)
	if err == ErrPlayerQuitException {
		return "", ErrPlayerQuitException
	}
	gamedata.UpdateEnemyBoard(input)
	fmt.Println("You're playing against: ", desc.Opponent, desc.Opp_desc)
	resp, err := Fire(c, input)
	if err != nil {
		return "", err
	}
	switch resp {
	case "miss":
		fmt.Println("miss")
		time.Sleep(1000 * time.Millisecond)
	case "hit":
		fmt.Println("hit")
	case "sunk":
		fmt.Println("You've sunk the ship! Congrats!")
	default:
		return "", err
	}
	return resp, nil
}

func GetPlayerInput(message string, shot bool) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	var input string
	var err error
	fmt.Println(message)
	for {
		input, err = reader.ReadString('\n')
		if err != nil {
			log.Println("getPlayerInput:", err)
			fmt.Println("Unexpected error when getting the input, try again")
			continue
		}
		input = strings.TrimSpace(input)
		if input == "quit" || input == "exit" {
			return "", ErrPlayerQuitException
		}
		if shot {
			err = validateShotCoords(input)
			if err != nil {
				log.Println("getPlayerInput:", err)
				fmt.Println(err)
				continue
			}
		}
		break
	}
	return input, nil
}

func validateShotCoords(coords string) error {
	if len(coords) < 2 {
		fmt.Println(ErrWrongCoordinates)
		return ErrWrongCoordinates
	}

	matched, err := regexp.MatchString("[A-J]", string(coords[0]))
	if !matched || err != nil {
		return ErrWrongCoordinates
	}

	if len(coords) == 2 {
		matched, err = regexp.MatchString("[0-9]", string(coords[1]))
		if !matched || err != nil {
			return ErrWrongCoordinates
		}
	}

	if len(coords) > 2 {
		matched, err = regexp.MatchString("10", coords[1:3])
		if !matched || err != nil {
			return ErrWrongCoordinates
		}
	}
	return nil
}
