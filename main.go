package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/s22678/statki/app"
	"github.com/s22678/statki/connect"
	"github.com/s22678/statki/gamedata"
	"github.com/s22678/statki/view"
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
	gameInitiated             = false
	withBot                   = false
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
		input, _ := app.GetPlayerInput("1) show players\n2) play with the bot\n3) play online\n4) display all players stats\n5) display single player stats\n6) display heatmap\n7) quit\n8) new gui prototype")
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
			withBot = true
			app.Play(withBot)
			gameInitiated = true
		case input == "3":
			withBot = false
			app.Play(withBot)
			gameInitiated = true
		case input == "4":
			stats, err := gamedata.GetAllPlayersStats()
			if err != nil {
				log.Println("get all players stats error:", err)
				continue
			}
			for _, stat := range stats {
				fmt.Println("Nick:", stat.Nick, "Games:", stat.Games, "Wins:", stat.Wins, "Rank:", stat.Rank, "Points", stat.Points)
			}
		case input == "5":
			input, _ := app.GetPlayerInput("get the stats for player:")
			stats, err := gamedata.GetOnePlayerStats(input)
			if err != nil {
				log.Println("get player", input, "stats error:", err)
				continue
			}
			fmt.Println("Nick:", stats.Nick, "Games:", stats.Games, "Wins:", stats.Wins, "Rank:", stats.Rank, "Points", stats.Points)
		case input == "6":
			gamedata.DisplayHeatMap()
		case input == "7":
			return
		case input == "8":
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			c := &connect.Connection{}
			err := c.InitGame(true, "", "", "", nil)
			if err != nil {
				log.Println("main:", err)
				return
			}

			statuschan := make(chan gamedata.StatusResponse)

			go func(statuschan chan gamedata.StatusResponse) {
				for {
					time.Sleep(1 * time.Second)
					select {
					case <-ctx.Done():
						return
					default:
						status, err := gamedata.GetStatus(c)
						if err != nil {
							log.Println("main:", err)
							return
						}
						statuschan <- *status
					}
				}
			}(statuschan)

			for {
				fmt.Println("waiting")
				status := <-statuschan

				if status.Should_fire {
					wg, err := view.NewGui(c)
					if err != nil {
						log.Println("main:", err)
						break
					}
					wg.Play()
					fmt.Println("done!")
					break
				}
			}

		default:
			continue
		}
		if gameInitiated {
			input, _ = app.GetPlayerInput("play again? [yes]/[no]")
			if input == "yes" {
				app.Play(withBot)
			}
			gameInitiated = false
		}
	}
}
