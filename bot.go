package stare

import (
	"context"

	"github.com/intrntsrfr/meido/pkg/mio"
	"github.com/intrntsrfr/meido/pkg/utils"
)

type Bot struct {
	Bot    *mio.Bot
	logger mio.Logger
	config *utils.Config
	db     DB
	store  *Store
}

func NewBot(config *utils.Config, db DB) *Bot {
	logger := newLogger("bot")

	b := mio.NewBotBuilder(config).
		WithDefaultHandlers().
		WithLogger(logger).
		Build()

	kvStore, err := NewStore(logger)
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

func (b *Bot) Run(ctx context.Context) error {
	b.registerModules()
	b.registerDiscordHandlers()
	b.registerMioHandlers()
	return b.Bot.Run(ctx)
}

func (b *Bot) Close() {
	b.Bot.Close()
}

func (b *Bot) registerModules() {
	modules := []mio.Module{
		NewModule(b.Bot, b.db, b.logger),
	}
	for _, mod := range modules {
		b.Bot.RegisterModule(mod)
	}
}

func (b *Bot) registerDiscordHandlers() {
	b.Bot.Discord.AddEventHandler(disconnectHandler(b))
	b.Bot.Discord.AddEventHandler(guildBanAddHandler(b))
	b.Bot.Discord.AddEventHandler(guildBanRemoveHandler(b))
	b.Bot.Discord.AddEventHandler(guildCreateHandler(b))
	b.Bot.Discord.AddEventHandler(guildMemberAddHandler(b))
	b.Bot.Discord.AddEventHandler(guildMemberRemoveHandler(b))
	b.Bot.Discord.AddEventHandler(guildMemberUpdateHandler(b))
	b.Bot.Discord.AddEventHandler(guildMembersChunkHandler(b))
	b.Bot.Discord.AddEventHandler(messageCreateHandler(b))
	b.Bot.Discord.AddEventHandler(messageDeleteBulkHandler(b))
	b.Bot.Discord.AddEventHandler(messageDeleteHandler(b))
	b.Bot.Discord.AddEventHandler(messageUpdateHandler(b))
}

func (b *Bot) registerMioHandlers() {
	b.Bot.AddHandler(logApplicationCommandPanicked(b))
	b.Bot.AddHandler(logApplicationCommandRan(b))
}
