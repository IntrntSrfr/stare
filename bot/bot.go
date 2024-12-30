package bot

import (
	"context"
	"fmt"
	"time"

	"github.com/intrntsrfr/functional-logger/database"
	"github.com/intrntsrfr/functional-logger/kvstore"
	"github.com/intrntsrfr/meido/pkg/mio"
	"github.com/intrntsrfr/meido/pkg/mio/bot"
	"github.com/intrntsrfr/meido/pkg/utils"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	Bot    *bot.Bot
	logger mio.Logger
	config *utils.Config
	db     database.DB
	store  *kvstore.Store
}

func New(config *utils.Config, db database.DB) *Bot {
	logger := newLogger("Bot")

	b := bot.NewBotBuilder(config).
		WithDefaultHandlers().
		WithLogger(logger).
		Build()

	kvStore, err := kvstore.NewStore(logger.log)
	if err != nil {
		panic("failed to create kvstore")
	}

	return &Bot{
		Bot:    b,
		db:     db,
		logger: logger,
		config: config,
		store:  kvStore,
	}
}

func (m *Bot) Run(ctx context.Context) error {
	m.registerModules()
	m.registerDiscordHandlers()
	return m.Bot.Run(ctx)
}

func (m *Bot) Close() {
	m.Bot.Close()
}

func (m *Bot) registerModules() {
	modules := []bot.Module{
		NewModule(m.Bot, m.db, m.logger),
	}
	for _, mod := range modules {
		m.Bot.RegisterModule(mod)
	}
}

func (m *Bot) registerDiscordHandlers() {
	m.Bot.Discord.AddEventHandlerOnce(statusLoop(m))
	m.Bot.Discord.AddEventHandler(disconnectHandler(m))
	m.Bot.Discord.AddEventHandler(guildBanAddHandler(m))
	m.Bot.Discord.AddEventHandler(guildBanRemoveHandler(m))
	m.Bot.Discord.AddEventHandler(guildCreateHandler(m))
	m.Bot.Discord.AddEventHandler(guildMemberAddHandler(m))
	m.Bot.Discord.AddEventHandler(guildMemberRemoveHandler(m))
	m.Bot.Discord.AddEventHandler(guildMemberUpdateHandler(m))
	m.Bot.Discord.AddEventHandler(guildMembersChunkHandler(m))
	m.Bot.Discord.AddEventHandler(messageCreateHandler(m))
	m.Bot.Discord.AddEventHandler(messageDeleteBulkHandler(m))
	m.Bot.Discord.AddEventHandler(messageDeleteHandler(m))
	m.Bot.Discord.AddEventHandler(messageUpdateHandler(m))
}

const totalStatusDisplays = 1

func statusLoop(m *Bot) func(s *discordgo.Session, r *discordgo.Ready) {
	statusTimer := time.NewTicker(time.Second * 15)
	return func(s *discordgo.Session, r *discordgo.Ready) {
		display := 0
		go func() {
			for range statusTimer.C {
				var (
					name       string
					statusType discordgo.ActivityType
				)
				switch display {
				case 0:
					srvCount := m.Bot.Discord.GuildCount()
					name = fmt.Sprintf("%v servers", srvCount)
					statusType = discordgo.ActivityTypeWatching
				}

				_ = s.UpdateStatusComplex(discordgo.UpdateStatusData{
					Activities: []*discordgo.Activity{{
						Name: name,
						Type: statusType,
					}},
				})
				display = (display + 1) % totalStatusDisplays
			}
		}()
	}
}
