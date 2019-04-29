package loggerbot

import (
	"database/sql"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/intrntsrfr/functional-logger/owo"

	"go.uber.org/zap"

	"github.com/bwmarrin/discordgo"
	"github.com/intrntsrfr/functional-logger/loggerdb"
)

type Bot struct {
	loggerDB  *loggerdb.DB
	logger    *zap.Logger
	db        *sql.DB
	client    *discordgo.Session
	config    *Config
	owo       *owo.OWOClient
	starttime time.Time
}

func NewLoggerBot(Config *Config, LoggerDB *loggerdb.DB, Log *zap.Logger) (*Bot, error) {

	client, err := discordgo.New("Bot " + Config.Token)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	Log.Info("created discord client")

	psql, err := sql.Open("postgres", Config.ConnectionString)
	if err != nil {
		fmt.Println("could not connect to db " + err.Error())
		Log.Error(err.Error())
		return nil, err
	}
	Log.Info("Established postgres connection")

	owoCl := owo.NewOWOClient(Config.OwoAPIKey)
	Log.Info("created owo client")

	return &Bot{
		loggerDB:  LoggerDB,
		logger:    Log,
		db:        psql,
		client:    client,
		config:    Config,
		owo:       owoCl,
		starttime: time.Now(),
	}, nil
}

func (b *Bot) Close() {
	b.client.Close()
}

func (b *Bot) Run() error {
	b.loggerDB.LoadTotal()

	b.addHandlers()
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	return b.client.Open()
}

func (b *Bot) addHandlers() {
	b.client.AddHandler(b.guildCreateHandler)
	b.client.AddHandler(b.guildMemberUpdateHandler)
	b.client.AddHandler(b.guildMemberAddHandler)
	b.client.AddHandler(b.guildMemberRemoveHandler)
	b.client.AddHandler(b.guildMembersChunkHandler)
	b.client.AddHandler(b.guildBanAddHandler)
	b.client.AddHandler(b.guildBanRemoveHandler)
	b.client.AddHandler(b.messageCreateHandler)
	b.client.AddHandler(b.messageUpdateHandler)
	b.client.AddHandler(b.messageDeleteHandler)
	b.client.AddHandler(b.messageDeleteBulkHandler)
	b.client.AddHandler(b.readyHandler)
	b.client.AddHandler(b.disconnectHandler)
}

func (b *Bot) guildUnavailableHandler(s *discordgo.Session, g *discordgo.GuildDelete) {

}

func (b *Bot) readyHandler(s *discordgo.Session, r *discordgo.Ready) {

	b.loggerDB.RunGC()

	timer := time.NewTicker(time.Hour)
	go func() {
		for range timer.C {
			b.loggerDB.RunGC()
		}
	}()

	fmt.Println(fmt.Sprintf("Logged in as %v.", r.User.String()))
}

func (b *Bot) disconnectHandler(s *discordgo.Session, d *discordgo.Disconnect) {
	atomic.StoreInt64(&b.loggerDB.TotalMembers, 0)
	fmt.Println("DISCONNECTED AT ", time.Now().Format(time.RFC1123))
}
