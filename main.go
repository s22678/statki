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

var (
	myNick = ""
)

func main() {
	f, err := os.OpenFile("game.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	c := connect.Connection{
		Url: url,
	}
	log.SetOutput(f)
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("podaj nicka (twojego)")
	myNick, err = reader.ReadString('\n')
	myNick = strings.TrimSpace(myNick)
	for {
		fmt.Println("1) pokaz graczy")
		fmt.Println("2) zagraj z botem")
		fmt.Println("3) zagraj online")
		fmt.Println("4) zakoncz")
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		switch {
		case strings.TrimSpace(input) == "1":
			players, err := c.ListPlayers()
			if err != nil {
				log.Println(err)
				continue
			}
			for _, v := range players {
				fmt.Println(v.Nick, v.Game_status)
			}
		case strings.TrimSpace(input) == "2":
			playNewGame(&c, reader, true)
		case strings.TrimSpace(input) == "3":

			playNewGame(&c, reader, false)
		case strings.TrimSpace(input) == "4":
			return
		}

	}
}
func playNewGame(c *connect.Connection, reader *bufio.Reader, withBot bool) {
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
	a.Board()
	for sr.Game_status != "ended" {
		if sr.Should_fire == true {
			a.UpdatePlayerBoard(sr.Opp_shots)
			fmt.Println("Nick przeciwnika: ", desc.Opponent, " Opis przeciwnika: ", desc.Opp_desc)
			resp, err := fire(reader, a, desc)
			if err != nil {
				log.Println("ERROR", err)
				break
			}
			for resp != "miss" {
				resp, err = fire(reader, a, desc)
				if err != nil {
					log.Println("ERROR", err)
					break
				}
			}
		} else {
			time.Sleep(1000 * time.Millisecond)
		}
		sr, err = c.Status()
		if sr.Last_game_status == "win" {
			fmt.Println("WYGRANA!")
		}
		if sr.Last_game_status == "lose" {
			fmt.Println("przegrana :(")
		}

		// fmt.Println("Desc:", sr.Desc, " Game_status:", sr.Game_status, " Last_game_status:", sr.Last_game_status, " Nick:", sr.Nick, " Opp_desc:", sr.Opp_desc, " Opp_shots:", sr.Opp_shots, " Opponent:", sr.Opponent, " Should_fire:", sr.Should_fire, " Timer:", sr.Timer)
	}
}

func fire(reader *bufio.Reader, a app.Application, desc *connect.StatusResponse) (string, error) {
	fmt.Println("twoj ruch!")
	input, err := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if err != nil {
		fmt.Println("ups!")
		return "", err
	}
	resp, _ := a.Fire(input)
	a.UpdateEnemyBoard(input)
	fmt.Println("Nick przeciwnika: ", desc.Opponent, " Opis przeciwnika: ", desc.Opp_desc)
	if resp == "miss" {
		fmt.Println("pudlo")
	} else {
		fmt.Println("trafiony")
	}
	time.Sleep(300 * time.Millisecond)
	return resp, nil
}
