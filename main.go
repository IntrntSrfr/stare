package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/intrntsrfr/functional-logger/loggerbot"
	"github.com/intrntsrfr/functional-logger/loggerdb"
)

func main() {
	jeff := zap.NewDevelopmentConfig()
	jeff.OutputPaths = []string{"./logs.txt"}
	jeff.ErrorOutputPaths = []string{"./logs.txt"}
	logger, err := jeff.Build()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer logger.Sync()

	logger.Info("logger construction succeeded")

	file, err := ioutil.ReadFile("./config.json")
	if err != nil {
		fmt.Printf("Config file not found.\nPlease press enter.")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
		return
	}

	var config loggerbot.Config
	json.Unmarshal(file, &config)

	loggerDB, err := loggerdb.NewDB(logger.Named("db"))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer loggerDB.Close()

	client, err := loggerbot.NewLoggerBot(&config, loggerDB, logger.Named("discord"))
	if err != nil {
		return
	}

	err = client.Run()
	if err != nil {
		return
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	client.Close()
}
