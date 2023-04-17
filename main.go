package main

import (
	"log"

	"github.com/s22678/statki/connect"
)

func main() {
	err := connect.InitGame()
	if err != nil {
		log.Fatal(err)
	}
}
