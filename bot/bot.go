package bot

import (
	"github.com/intrntsrfr/functional-logger/database"
	"github.com/intrntsrfr/functional-logger/discord"
	"github.com/intrntsrfr/functional-logger/kvstore"
	"time"

	"github.com/intrntsrfr/owo"

	"go.uber.org/zap"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	store     *kvstore.Store
	log       *zap.Logger
	db        database.DB
	disc      *discord.Discord
	sess      *discordgo.Session
	config    *Config
	owo       *owo.Client
	startTime time.Time
}

type Config struct {
	Store *kvstore.Store
	Log   *zap.Logger
	DB    database.DB
	Owo   *owo.Client
	Token string
}

func NewBot(c *Config) (*Bot, error) {
	b := &Bot{
		store:     c.Store,
		log:       c.Log,
		db:        c.DB,
		config:    c,
		owo:       c.Owo,
		startTime: time.Now(),
	}

	disc, err := discord.NewDiscord(c.Token, c.Log)
	if err != nil {
		return nil, err
	}
	b.disc = disc
	b.sess = disc.Sess
	/*
		disc.AddHandler(b.guildCreateHandler)
		disc.AddHandler(b.guildMemberUpdateHandler)
		disc.AddHandler(b.guildMemberAddHandler)
		disc.AddHandler(b.guildMemberRemoveHandler)
		disc.AddHandler(b.guildMembersChunkHandler)
		disc.AddHandler(b.guildBanAddHandler)
		disc.AddHandler(b.guildBanRemoveHandler)
		disc.AddHandler(b.messageCreateHandler)
		disc.AddHandler(b.messageUpdateHandler)
		disc.AddHandler(b.messageDeleteHandler)
		disc.AddHandler(b.messageDeleteBulkHandler)
		disc.AddHandler(b.readyHandler)
		disc.AddHandler(b.disconnectHandler)
	*/
	return b, nil
}

func (b *Bot) Close() {
	b.disc.Close()
}

func (b *Bot) Run() error {
	go b.listen(b.disc.Events)

	err := b.disc.Open()
	if err != nil {
		return err
	}
	return nil
}

func (b *Bot) listen(evtCh <-chan interface{}) {
	for {
		evt := <-evtCh

		switch evt.(type) {
		case *discordgo.MessageCreate:
			m, _ := evt.(*discordgo.MessageCreate)
			b.log.Info("new message", zap.String("content", m.Content))

		}
	}
}

func handleMessageCreate() {

}
