package bot

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
}

func NewModule(b *bot.Bot, logger mio.Logger) *module {
	logger = logger.Named("commands")
	return &module{
		ModuleBase: bot.NewModule(b, "commands", logger),
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
		newViewSettingsSlash(m),
		newSetSettingsSlash(m),
	); err != nil {
		return err
	}

	return nil
}

func newHelpSlash(m *module) *bot.ModuleApplicationCommand {
	cmd := bot.NewModuleApplicationCommandBuilder(m, "help").
		Type(discordgo.ChatApplicationCommand).
		Description("Get help with the bot")

	run := func(d *discord.DiscordApplicationCommand) {
		text := strings.Builder{}
		text.WriteString("To set a log channel, use the `/set` command with the following options:\n\n")
		text.WriteString("Logtypes:\n")
		text.WriteString("`join` - When a user joins the server\n")
		text.WriteString("`leave` - When a user leaves the server\n")
		text.WriteString("`msgdelete` - When a message is deleted\n")
		text.WriteString("`msgedit` - When a message is edited\n")
		text.WriteString("`ban` - When a user is banned\n")
		text.WriteString("`unban` - When a user is unbanned\n")
		text.WriteString("\n")
		text.WriteString("Example - `/set logtype:join`\n")
		text.WriteString("Example - `/set logtype:join channel:#join-logs`\n")
		text.WriteString("Example - `/set logtype:join channel:1234123412341234`\n")

		d.Respond(text.String())
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
			AddField("Running since", fmt.Sprintf("<t:%v:R>", m.startTime.Unix()), false)

		d.RespondEmbed(embed.Build())
	}

	return cmd.Execute(run).Build()
}

func newViewSettingsSlash(m *module) *bot.ModuleApplicationCommand {
	cmd := bot.NewModuleApplicationCommandBuilder(m, "settings").
		Type(discordgo.ChatApplicationCommand).
		Description("View the current settings")

	run := func(d *discord.DiscordApplicationCommand) {
		d.Respond("Settings")
	}

	return cmd.Execute(run).Build()
}

func newSetSettingsSlash(m *module) *bot.ModuleApplicationCommand {
	// old cmd
	/*

		args := strings.Fields(m.Content)
		uperms, err := s.State.UserChannelPermissions(m.Author.ID, ch.ID)
		if err != nil {
			return
		}

		if args[0] == "fl.set" {
			if len(args) < 2 {
				return
			}

			gc, err := b.db.GetGuild(g.ID)
			if err != nil {
				b.logger.Error("failed to get guild config", zap.Error(err))
				return
			}

			if uperms&(discordgo.PermissionAdministrator|discordgo.PermissionAll) == 0 {
				_, _ = s.ChannelMessageSend(ch.ID, "This is admin only, sorry!")
				return
			}

			setChannel := ch
			if len(args) >= 3 {
				chStr := utils.TrimChannelID(args[2])
				setChannel, err = s.State.Channel(chStr)
				if err != nil || setChannel.GuildID != g.ID {
					b.logger.Debug("failed to get guild", zap.Error(err))
					return
				}
			}
			switch strings.ToLower(args[1]) {
			case "join":
				gc.JoinLog = setChannel.ID
			case "leave":
				gc.LeaveLog = setChannel.ID
			case "msgdelete":
				gc.MsgDeleteLog = setChannel.ID
			case "msgedit":
				gc.MsgEditLog = setChannel.ID
			case "ban":
				gc.BanLog = setChannel.ID
			case "unban":
				gc.UnbanLog = setChannel.ID
			}

			err = b.db.UpdateGuild(g.ID, gc)
			if err != nil {
				_, _ = s.ChannelMessageSend(ch.ID, "Could not update config ")
				b.logger.Error("failed to update guild config", zap.Error(err))
				return
			}

			_, _ = s.ChannelMessageSend(ch.ID, "Updated config")
		}

	*/

	cmd := bot.NewModuleApplicationCommandBuilder(m, "set").
		Type(discordgo.ChatApplicationCommand).
		Description("Set a setting")

	run := func(d *discord.DiscordApplicationCommand) {
		d.Respond("Set")
	}

	return cmd.Execute(run).Build()
}
