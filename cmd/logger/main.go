package main

import (
	"encoding/json"
	"github.com/intrntsrfr/functional-logger/database"
	"github.com/intrntsrfr/functional-logger/kvstore"
	"go.uber.org/zap/zapcore"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/intrntsrfr/functional-logger/bot"
	"github.com/intrntsrfr/owo"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
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

	// dependencies
	z := zap.NewDevelopmentConfig()
	z.OutputPaths = []string{"stdout", "./logs.txt"}
	z.ErrorOutputPaths = []string{"stderr", "./logs.txt"}
	z.Level.SetLevel(zapcore.WarnLevel)
	logger, err := z.Build()
	if err != nil {
		log.Fatalf("failed to build logger: %v", err)
	}
	defer logger.Sync()

	psql, err := database.NewJsonDatabase("./data.json")
	if err != nil {
		log.Fatalf("failed to open DB connection: %v", err)
	}
	defer psql.Close()

	owoCl := owo.NewClient(config.OwoAPIKey)
	store, err := kvstore.NewStore(logger.Named("store"))
	if err != nil {
		log.Fatalf("failed to open kv store: %v", err)
	}
	defer store.Close()

	// bot
	client, err := bot.NewBot(&bot.Config{
		Store: store,
		Log:   logger.Named("bot"),
		DB:    psql,
		Owo:   owoCl,
		Token: config.Token,
	})
	if err != nil {
		log.Fatalf("failed to create bot: %v", err)
	}
	defer client.Close()

	// run
	err = client.Run()
	if err != nil {
		log.Fatalf("failed to run: %v", err)
	}

	// block until ctrl-c
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	<-sc
}
