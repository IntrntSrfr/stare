package bot

import (
	"fmt"
	"github.com/intrntsrfr/functional-logger/database"
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

	s, err := discordgo.New("Bot " + c.Token)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	b.sess = s

	return b, nil
}

func (b *Bot) Close() {
	b.log.Info("Shutting down bot.")
	b.db.Close()
	b.sess.Close()
}

func (b *Bot) Run() error {

	b.addHandlers()
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	return b.sess.Open()
}

func (b *Bot) addHandlers() {
	b.sess.AddHandler(b.guildCreateHandler)
	b.sess.AddHandler(b.guildMemberUpdateHandler)
	b.sess.AddHandler(b.guildMemberAddHandler)
	b.sess.AddHandler(b.guildMemberRemoveHandler)
	b.sess.AddHandler(b.guildMembersChunkHandler)
	b.sess.AddHandler(b.guildBanAddHandler)
	b.sess.AddHandler(b.guildBanRemoveHandler)
	b.sess.AddHandler(b.messageCreateHandler)
	b.sess.AddHandler(b.messageUpdateHandler)
	b.sess.AddHandler(b.messageDeleteHandler)
	b.sess.AddHandler(b.messageDeleteBulkHandler)
	b.sess.AddHandler(b.readyHandler)
	b.sess.AddHandler(b.disconnectHandler)
}
