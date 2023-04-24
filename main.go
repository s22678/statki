package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/s22678/statki/app"
	"github.com/s22678/statki/connect"
)

const (
	url = "https://go-pjatk-server.fly.dev"
)

func main() {
	f, err := os.OpenFile("game.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)
	reader := bufio.NewReader(os.Stdin)
	// for {
	// 	fmt.Println("1) pokaz graczy")
	// 	fmt.Println("2) zagraj z botem")
	// 	input, err := reader.ReadString('\n')
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	switch {
	// 	case strings.TrimSpace(input) == "1":

	// 	}

	// }
	c := connect.Connection{
		Url: url,
	}
	err = c.InitGame()
	if err != nil {
		log.Fatal(err)
	}
	a := app.Application{
		Con: c,
	}

	sr, err := c.Status()
	if err != nil {
		log.Println("ERROR getting a game satus, exiting")
		return
	}
	for !sr.Should_fire {
		sr, _ = c.Status()
	}
	a.Board()
	for sr.Game_status != "ended" {
		if sr.Should_fire == true {
			a.UpdatePlayerBoard(sr.Opp_shots)
			resp, err := fire(reader, a)
			if err != nil {
				log.Println("ERROR", err)
				break
			}
			for resp != "miss" {
				resp, err = fire(reader, a)
				if err != nil {
					log.Println("ERROR", err)
					break
				}
			}
		} else {
			time.Sleep(1000 * time.Millisecond)
		}
		sr, err = c.Status()

		// fmt.Println("Desc:", sr.Desc, " Game_status:", sr.Game_status, " Last_game_status:", sr.Last_game_status, " Nick:", sr.Nick, " Opp_desc:", sr.Opp_desc, " Opp_shots:", sr.Opp_shots, " Opponent:", sr.Opponent, " Should_fire:", sr.Should_fire, " Timer:", sr.Timer)
	}
}

func fire(reader *bufio.Reader, a app.Application) (string, error) {
	fmt.Println("twoj ruch!")
	input, err := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if err != nil {
		fmt.Println("ups!")
		return "", err
	}
	resp, _ := a.Fire(input)
	if resp == "miss" {
		fmt.Println("pudlo")
	} else {
		fmt.Println("trafiony")
	}
	a.UpdateEnemyBoard(input)
	return resp, nil
}
