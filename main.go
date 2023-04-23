package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/s22678/statki/app"
	"github.com/s22678/statki/connect"
)

const (
	url = "https://go-pjatk-server.fly.dev"
)

func main() {
	f, err := os.OpenFile("testlogfile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)
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

	for {
		fmt.Println(c.Status())
		a.Board()
		time.Sleep(1 * time.Second)
	}
}
