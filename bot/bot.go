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
		ctx := &Context{
			b:  b,
			s:  b.sess,
			gc: nil,
		}

		if e, ok := evt.(*discordgo.Ready); ok {
			go readyHandler(ctx, e)
		} else if e, ok := evt.(*discordgo.Disconnect); ok {
			go disconnectHandler(ctx, e)
			/*
				} else if e, ok := evt.(*discordgo.MessageDeleteBulk); ok {
					gc, err := b.db.GetGuild(e.GuildID)
					if err != nil {
						continue
					}
					ctx.gc = gc
				} else if e, ok := evt.(*discordgo.MessageDelete); ok {
					gc, err := b.db.GetGuild(e.GuildID)
					if err != nil {
						continue
					}
					ctx.gc = gc

				} else if e, ok := evt.(*discordgo.MessageUpdate); ok {
					gc, err := b.db.GetGuild(e.GuildID)
					if err != nil {
						continue
					}
					ctx.gc = gc
			*/

		} else if e, ok := evt.(*discordgo.MessageCreate); ok {
			gc, err := b.db.GetGuild(e.GuildID)
			if err != nil {
				continue
			}
			ctx.gc = gc

			go messageCreateHandler(ctx, e)
		} else if e, ok := evt.(*discordgo.GuildBanRemove); ok {
			gc, err := b.db.GetGuild(e.GuildID)
			if err != nil || gc.UnbanLog == "" {
				continue
			}
			ctx.gc = gc

			go guildBanRemoveHandler(ctx, e)
		} else if e, ok := evt.(*discordgo.GuildBanAdd); ok {
			gc, err := b.db.GetGuild(e.GuildID)
			if err != nil || gc.BanLog == "" {
				continue
			}
			ctx.gc = gc

			go guildBanAddHandler(ctx, e)
		} else if e, ok := evt.(*discordgo.GuildMembersChunk); ok {
			gc, err := b.db.GetGuild(e.GuildID)
			if err != nil {
				continue
			}
			ctx.gc = gc

			go guildMembersChunkHandler(ctx, e)
		} else if e, ok := evt.(*discordgo.GuildMemberRemove); ok {
			gc, err := b.db.GetGuild(e.GuildID)
			if err != nil || gc.LeaveLog == "" {
				continue
			}
			ctx.gc = gc

			go guildMemberRemoveHandler(ctx, e)
		} else if e, ok := evt.(*discordgo.GuildMemberAdd); ok {
			gc, err := b.db.GetGuild(e.GuildID)
			if err != nil || gc.JoinLog == "" {
				continue
			}
			ctx.gc = gc

			go guildMemberAddHandler(ctx, e)
		} else if e, ok := evt.(*discordgo.GuildMemberUpdate); ok {
			gc, err := b.db.GetGuild(e.GuildID)
			if err != nil {
				continue
			}
			ctx.gc = gc

			go guildMemberUpdateHandler(ctx, e)
		} else if e, ok := evt.(*discordgo.GuildCreate); ok {
			go guildCreateHandler(ctx, e)
		}
	}
}
