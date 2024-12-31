package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"

	"github.com/intrntsrfr/meido/pkg/utils"
	"github.com/intrntsrfr/stare"
)

func main() {
	cfg := utils.NewConfig()
	loadConfig(cfg, "./config.json")

	jsonDb, err := stare.NewJsonDatabase("./data.json")
	if err != nil {
		panic(err)
	}
	defer jsonDb.Close()

	bot := stare.NewBot(cfg, jsonDb)
	defer bot.Close()

	if err := bot.Run(context.Background()); err != nil {
		panic(err)
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc
}

type config struct {
	Token  string `json:"token"`
	Shards int    `json:"shards"`
}

func loadConfig(cfg *utils.Config, path string) {
	f, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	var c config
	if err := json.Unmarshal(f, &c); err != nil {
		panic(err)
	}

	cfg.Set("token", c.Token)
	cfg.Set("shards", c.Shards)
}
