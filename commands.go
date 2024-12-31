package stare

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/intrntsrfr/meido/pkg/mio"
	"github.com/intrntsrfr/meido/pkg/mio/bot"
	"github.com/intrntsrfr/meido/pkg/mio/discord"
	"github.com/intrntsrfr/meido/pkg/utils/builders"
)

type module struct {
	*bot.ModuleBase
	startTime time.Time
	db        DB
}

func NewModule(b *bot.Bot, db DB, logger mio.Logger) *module {
	logger = logger.Named("commands")
	return &module{
		ModuleBase: bot.NewModule(b, "commands", logger),
		db:         db,
		startTime:  time.Now(),
	}
}

func (m *module) Hook() error {
	if err := m.RegisterCommands(); err != nil {
		return err
	}
	if err := m.RegisterApplicationCommands(
		newInfoSlash(m),
		newHelpSlash(m),
		newSettingsSlash(m),
	); err != nil {
		return err
	}

	return nil
}

func newHelpSlash(m *module) *bot.ModuleApplicationCommand {
	cmd := bot.NewModuleApplicationCommandBuilder(m, "help").
		Type(discordgo.ChatApplicationCommand).
		Description("Get help on how to use the bot")

	run := func(d *discord.DiscordApplicationCommand) {
		text := strings.Builder{}
		text.WriteString("What gets logged:\n")
		text.WriteString("1. When a user joins the server\n")
		text.WriteString("1. When a user leaves the server\n")
		text.WriteString("1. When a message is deleted\n")
		text.WriteString("1. When messages are bulk deleted\n")
		text.WriteString("1. When a message is edited\n")
		text.WriteString("1. When a user is banned\n")
		text.WriteString("1. When a user is unbanned\n")
		text.WriteString("\n")
		text.WriteString("To view the current settings, use the `/settings view` command\n")
		text.WriteString("To set a log channel, use the `/settings set` command\n")
		text.WriteString("\n")

		embed := builders.NewEmbedBuilder().
			WithTitle("Help").
			WithOkColor().
			WithDescription(text.String())
		d.RespondEmbed(embed.Build())
	}

	return cmd.Execute(run).Build()
}

func newInfoSlash(m *module) *bot.ModuleApplicationCommand {
	cmd := bot.NewModuleApplicationCommandBuilder(m, "info").
		Type(discordgo.ChatApplicationCommand).
		Description("Get information about the bot")

	run := func(d *discord.DiscordApplicationCommand) {
		embed := builders.NewEmbedBuilder().
			WithTitle("Info").
			WithOkColor().
			AddField("Golang version", runtime.Version(), false).
			AddField("Running since", fmt.Sprintf("<t:%v:R>", m.startTime.Unix()), false).
			AddField("Total guilds", fmt.Sprintf("%v", d.Discord.GuildCount()), false)
		d.RespondEmbed(embed.Build())
	}

	return cmd.Execute(run).Build()
}

func newSettingsSlash(m *module) *bot.ModuleApplicationCommand {
	logTypes := map[string]string{
		"join":      "User Join",
		"leave":     "User Leave",
		"msgdelete": "Message Delete",
		"msgedit":   "Message Edit",
		"ban":       "User Ban",
		"unban":     "User Unban",
	}

	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0, len(logTypes))
	for k, v := range logTypes {
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  v,
			Value: k,
		})
	}

	cmd := bot.NewModuleApplicationCommandBuilder(m, "settings").
		Type(discordgo.ChatApplicationCommand).
		Description("View or set the current settings").
		NoDM().
		Permissions(discordgo.PermissionAdministrator).
		AddSubcommand(&discordgo.ApplicationCommandOption{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "view",
			Description: "View the current settings",
		}).
		AddSubcommand(&discordgo.ApplicationCommandOption{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "set",
			Description: "Set a setting",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "type",
					Description: "The type of log to set",
					Required:    true,
					Choices:     choices,
				},
				{
					Type:        discordgo.ApplicationCommandOptionChannel,
					Name:        "channel",
					Description: "The channel to set the log to",
					Required:    true,
				},
			},
		})

	run := func(d *discord.DiscordApplicationCommand) {
		gc, err := m.db.GetGuild(d.GuildID())
		if err != nil {
			d.Respond("Failed to get guild config")
			return
		}

		if _, ok := d.Options("view"); ok {
			d.RespondEmbed(generateLogSettingsEmbed(gc))
			return
		} else if _, ok := d.Options("set"); ok {
			logType, ok := d.Options("set:type")
			if !ok {
				d.Respond("Log type not found")
				return
			}
			logTypeStr := logType.StringValue()

			chOpt, ok := d.Options("set:channel")
			if !ok {
				d.Respond("Channel not found")
				return
			}
			ch := chOpt.ChannelValue(d.Sess.Real())

			if ch == nil {
				d.Respond("Channel not found")
				return
			}

			switch logTypeStr {
			case "join":
				gc.JoinLog = ch.ID
			case "leave":
				gc.LeaveLog = ch.ID
			case "msgdelete":
				gc.MsgDeleteLog = ch.ID
			case "msgedit":
				gc.MsgEditLog = ch.ID
			case "ban":
				gc.BanLog = ch.ID
			case "unban":
				gc.UnbanLog = ch.ID
			}

			if err := m.db.UpdateGuild(d.GuildID(), gc); err != nil {
				d.Respond("Failed to update server config")
				return
			}

			embed := generateLogSettingsEmbed(gc)
			embed.Title = "Updated settings"

			resp := &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{embed},
				Flags:  discordgo.MessageFlagsEphemeral,
			}

			d.RespondComplex(resp, discordgo.InteractionResponseChannelMessageWithSource)
			return
		}
	}

	return cmd.Execute(run).Build()
}

func generateLogSettingsEmbed(gc *Guild) *discordgo.MessageEmbed {
	embed := builders.NewEmbedBuilder().
		WithTitle("Settings").
		WithOkColor().
		AddField("Join log", fmt.Sprintf("<#%v>", gc.JoinLog), true).
		AddField("Leave log", fmt.Sprintf("<#%v>", gc.LeaveLog), true).
		AddField("Message delete log", fmt.Sprintf("<#%v>", gc.MsgDeleteLog), true).
		AddField("Message edit log", fmt.Sprintf("<#%v>", gc.MsgEditLog), true).
		AddField("Ban log", fmt.Sprintf("<#%v>", gc.BanLog), true).
		AddField("Unban log", fmt.Sprintf("<#%v>", gc.UnbanLog), true)

	return embed.Build()
}
