package main

import (
	"encoding/json"
	"github.com/intrntsrfr/functional-logger/bot"
	"go.uber.org/zap"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"
)

type Config struct {
	Token            string `json:"token"`
	ConnectionString string `json:"connection_string"`
	OwoAPIKey        string `json:"owo_api_key"`
}

func main() {
	d, err := os.ReadFile("./config.json")
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}

	var config *Config
	err = json.Unmarshal(d, &config)
	if err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}

	z, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}

	b, err := bot.NewBot(&bot.Config{
		Store: nil,
		Log:   z.Named("bot"),
		DB:    nil,
		Owo:   nil,
		Token: config.Token,
	})
	defer b.Close()

	err = b.Run()
	if err != nil {
		panic(err)
	}

	// block until ctrl-c
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	<-sc
}
