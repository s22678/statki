package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/s22678/statki/app"
	"github.com/s22678/statki/connect"
	"github.com/s22678/statki/gamedata"
)

const (
	LogPath = "game.log"
)

var (
	ErrWrongResponseException = errors.New("there was an error in the response")
	ErrPlayerQuitException    = errors.New("player quit the game")
	ErrWrongCoordinates       = errors.New("coordinates did not match pattern [A-J]([0-9]|10), try again with correct coordinates")
	LogFile                   *os.File
	LogErr                    error
)

func init() {
	LogFile, LogErr = os.OpenFile("game.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if LogErr != nil {
		log.Fatalf("error opening file: %v", LogErr)
	}
	log.SetOutput(LogFile)
}

func main() {
	defer LogFile.Close()
	for {
		input, _ := getPlayerInput("1) show players\n2) play with the bot\n3) play online\n4) quit", false)
		switch {
		case input == "1":
			players, err := gamedata.ListPlayers()
			if err != nil {
				log.Println(err)
				continue
			}
			for _, v := range players {
				fmt.Println(v.Nick, v.Game_status)
			}
		case input == "2":
			c := connect.Connection{}
			playNewGame(&c, true)
		case strings.TrimSpace(input) == "3":
			c := connect.Connection{}
			playNewGame(&c, false)
		case input == "4":
			return
		default:
			continue
		}
	}
}

func playNewGame(c *connect.Connection, withBot bool) {
	var oppShotsDiff []string
	myNick, _ := getPlayerInput("set your nickname!", false)
	err := c.InitGame(withBot, "", myNick)
	if err != nil {
		log.Fatal(err)
	}
	a := app.Application{
		Con: *c,
	}

	sr, err := c.Status()
	if !withBot {
		for sr.Game_status == "waiting" {
			fmt.Println(sr.Game_status)
			time.Sleep(1 * time.Second)
		}
	}
	if err != nil {
		log.Println("ERROR getting a game satus, exiting")
		return
	}
	for !sr.Should_fire {
		sr, _ = c.Status()
		time.Sleep(300 * time.Millisecond)
	}
	desc, err := a.GetDescritpion()
	if err != nil {
		log.Println(err)
	}
	gamedata.GetBoard(c)
gameloop:
	for sr.Game_status != "ended" {
		var resp string
		if sr.Should_fire {
			oppShotsDiff = sr.Opp_shots[len(oppShotsDiff):len(sr.Opp_shots)]
			log.Printf("DUDE: %s", oppShotsDiff)
			gamedata.UpdatePlayerBoard(oppShotsDiff)
			fmt.Println("You're playing against: ", desc.Opponent, desc.Opp_desc)
			resp, err = fire(a, desc)
			if err == ErrPlayerQuitException {
				log.Println(ErrPlayerQuitException)
				a.QuitGame()
				break
			}
			for resp == "hit" || resp == "sunk" {
				resp, err = fire(a, desc)
				if err == ErrPlayerQuitException {
					log.Println(ErrPlayerQuitException)
					a.QuitGame()
					break gameloop
				}
			}
		} else {
			time.Sleep(1000 * time.Millisecond)
		}
		sr, err = c.Status()
		if err != nil {
			log.Println("ERROR", err)
			break
		}
		if sr.Last_game_status == "win" {
			fmt.Println("WYGRANA!")
		}
		if sr.Last_game_status == "lose" {
			fmt.Println("przegrana :(")
		}
		if sr.Last_game_status == "session not found" {
			fmt.Println("przegrano walkowerem")
		}

		gamedata.UpdatePlayerBoard(oppShotsDiff)
		gamedata.UpdateEnemyBoard(resp)
		fmt.Println("You're playing against: ", desc.Opponent, desc.Opp_desc)

		// fmt.Println("Desc:", sr.Desc, " Game_status:", sr.Game_status, " Last_game_status:", sr.Last_game_status, " Nick:", sr.Nick, " Opp_desc:", sr.Opp_desc, " Opp_shots:", sr.Opp_shots, " Opponent:", sr.Opponent, " Should_fire:", sr.Should_fire, " Timer:", sr.Timer)
	}
}

func fire(a app.Application, desc *connect.StatusResponse) (string, error) {
	input, err := getPlayerInput("your move!", true)
	if err == ErrPlayerQuitException {
		return "", ErrPlayerQuitException
	}
	gamedata.UpdateEnemyBoard(input)
	fmt.Println("Nick przeciwnika: ", desc.Opponent, " Opis przeciwnika: ", desc.Opp_desc)
	resp, err := a.Fire(input)
	switch resp {
	case "miss":
		fmt.Println("miss")
	case "hit":
		fmt.Println("hit")
	case "sunk":
		fmt.Println("You've sunk the ship! Congrats!")
	default:
		return "", err
	}
	time.Sleep(300 * time.Millisecond)
	return resp, nil
}

func getPlayerInput(message string, shot bool) (string, error) {
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
