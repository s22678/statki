package connect

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// func InitGame() error
// func Board() ([]string, error)
// func Status() (*StatusResponse, error)
// func Fire(coord string) (string, error)

var (
	gameData = map[string]interface{}{
		"coords":      nil,
		"desc":        "",
		"nick":        "",
		"target_nick": "",
		"wpbot":       true,
	}
)

func InitGame() error {
	b, err := json.Marshal(gameData)
	if err != nil {
		log.Println(err)
	}

	reader := bytes.NewReader(b)
	log.Println(string(b))
	resp, err := http.Post("https://go-pjatk-server.fly.dev/api/game", "application/json", reader)
	if err != nil {
		log.Println(resp)
		log.Println(err)
	}
	fmt.Println(resp.Header.Get("X-Auth-Token"))
	return err
}
