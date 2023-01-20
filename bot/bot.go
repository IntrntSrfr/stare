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

	disc, err := discord.NewDiscord(c.Token, c.Log.Named("discord"))
	if err != nil {
		return nil, err
	}
	b.disc = disc
	b.sess = disc.Sess

	return b, nil
}

func (b *Bot) Close() {
	b.log.Info("stopping bot")
	b.disc.Close()
}

func (b *Bot) Run() error {
	b.log.Info("starting bot")
	go b.listen(b.disc.Events)

	err := b.disc.Open()
	if err != nil {
		return err
	}
	return nil
}

func (b *Bot) listen(evtCh <-chan interface{}) {
	b.log.Info("listening for events")
	for {
		evt := <-evtCh
		ctx := &Context{
			b: b,
			s: b.sess,
			d: b.disc,
		}

		if e, ok := evt.(*discordgo.Ready); ok {
			b.log.Info("new event", zap.String("event", "ready"))
			go readyHandler(ctx, e)
		} else if e, ok := evt.(*discordgo.Disconnect); ok {
			b.log.Info("new event", zap.String("event", "disconnect"))
			go disconnectHandler(ctx, e)
		} else if e, ok := evt.(*discordgo.MessageDeleteBulk); ok {
			b.log.Info("new event", zap.String("event", "message delete bulk"))
			gc, err := b.db.GetGuild(e.GuildID)
			if err != nil {
				continue
			}
			ctx.gc = gc
			go messageDeleteBulkHandler(ctx, e)
		} else if e, ok := evt.(*discordgo.MessageDelete); ok {
			b.log.Info("new event", zap.String("event", "message delete"))
			gc, err := b.db.GetGuild(e.GuildID)
			if err != nil {
				continue
			}
			ctx.gc = gc
			go messageDeleteHandler(ctx, e)
		} else if e, ok := evt.(*discordgo.MessageUpdate); ok {
			b.log.Info("new event", zap.String("event", "message update"))
			gc, err := b.db.GetGuild(e.GuildID)
			if err != nil {
				continue
			}
			ctx.gc = gc
			go messageUpdateHandler(ctx, e)
		} else if e, ok := evt.(*discordgo.MessageCreate); ok {
			b.log.Info("new event", zap.String("event", "message create"))
			gc, err := b.db.GetGuild(e.GuildID)
			if err != nil {
				continue
			}
			ctx.gc = gc
			go messageCreateHandler(ctx, e)
		} else if e, ok := evt.(*discordgo.GuildBanRemove); ok {
			b.log.Info("new event", zap.String("event", "guild ban remove"))
			gc, err := b.db.GetGuild(e.GuildID)
			if err != nil || gc.UnbanLog == "" {
				continue
			}
			ctx.gc = gc
			go guildBanRemoveHandler(ctx, e)
		} else if e, ok := evt.(*discordgo.GuildBanAdd); ok {
			b.log.Info("new event", zap.String("event", "guild ban add"))
			gc, err := b.db.GetGuild(e.GuildID)
			if err != nil || gc.BanLog == "" {
				continue
			}
			ctx.gc = gc
			go guildBanAddHandler(ctx, e)
		} else if e, ok := evt.(*discordgo.GuildMembersChunk); ok {
			b.log.Info("new event", zap.String("event", "guild members chunk"))
			gc, err := b.db.GetGuild(e.GuildID)
			if err != nil {
				continue
			}
			ctx.gc = gc
			go guildMembersChunkHandler(ctx, e)
		} else if e, ok := evt.(*discordgo.GuildMemberRemove); ok {
			b.log.Info("new event", zap.String("event", "guild member remove"))
			gc, err := b.db.GetGuild(e.GuildID)
			if err != nil || gc.LeaveLog == "" {
				continue
			}
			ctx.gc = gc
			go guildMemberRemoveHandler(ctx, e)
		} else if e, ok := evt.(*discordgo.GuildMemberAdd); ok {
			b.log.Info("new event", zap.String("event", "guild member add"))
			gc, err := b.db.GetGuild(e.GuildID)
			if err != nil || gc.JoinLog == "" {
				continue
			}
			ctx.gc = gc
			go guildMemberAddHandler(ctx, e)
		} else if e, ok := evt.(*discordgo.GuildMemberUpdate); ok {
			b.log.Info("new event", zap.String("event", "guild member update"))
			gc, err := b.db.GetGuild(e.GuildID)
			if err != nil {
				continue
			}
			ctx.gc = gc
			go guildMemberUpdateHandler(ctx, e)
		} else if e, ok := evt.(*discordgo.GuildCreate); ok {
			b.log.Info("new event", zap.String("event", "guild create"))
			go guildCreateHandler(ctx, e)
		}
	}
}
