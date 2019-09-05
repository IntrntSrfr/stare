package loggerbot

import (
	"fmt"
	"strconv"
	"time"

	"github.com/intrntsrfr/owo"

	"go.uber.org/zap"

	"github.com/bwmarrin/discordgo"
	"github.com/intrntsrfr/functional-logger/loggerdb"
	"github.com/jmoiron/sqlx"
)

type Bot struct {
	loggerDB  *loggerdb.DB
	logger    *zap.Logger
	db        *sqlx.DB
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

	psql, err := sqlx.Connect("postgres", Config.ConnectionString)
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
	b.logger.Info("Shutting down bot.")
	b.db.Close()
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

	gctimer := time.NewTicker(time.Hour)
	statustimer := time.NewTicker(time.Second * 15)
	go func() {
		for range gctimer.C {
			b.loggerDB.RunGC()
		}
	}()

	go func() {
		i := 0
		for range statustimer.C {
			switch i {
			case 0:
				s.UpdateStatus(0, "fl.help")
				i++
			case 1:
				m := b.loggerDB.TotalMembers
				s.UpdateStatusComplex(discordgo.UpdateStatusData{
					Game: &discordgo.Game{
						Name: "over all " + strconv.FormatInt(m, 10) + " of you",
						Type: discordgo.GameTypeWatching,
					},
				})
				i++
			default:
				i = 0
			}

		}
	}()

	fmt.Println(fmt.Sprintf("Logged in as %v.", r.User.String()))
}

func (b *Bot) disconnectHandler(s *discordgo.Session, d *discordgo.Disconnect) {
	//atomic.StoreInt64(&b.loggerDB.TotalMembers, 0)
	fmt.Println("DISCONNECTED AT ", time.Now().Format(time.RFC1123))
}
