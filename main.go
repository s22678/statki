package main

import (
	"log"

	"github.com/s22678/statki/connect"
)

const (
	url = "https://go-pjatk-server.fly.dev"
)

func main() {
	c := connect.Connection{
		Url: url,
	}
	err := c.InitGame()
	if err != nil {
		log.Fatal(err)
	}
	c.Status()
}
