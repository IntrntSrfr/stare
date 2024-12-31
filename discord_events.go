package stare

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dgraph-io/badger"
	"github.com/intrntsrfr/meido/pkg/utils"
	"github.com/intrntsrfr/meido/pkg/utils/builders"
	"go.uber.org/zap"
)

type Color int

const (
	ColorRed    Color = 0xff0000
	ColorGreen  Color = 0x00ff00
	ColorBlue   Color = 0x61d1ed
	ColorWhite  Color = 0xffffff
	ColorOrange Color = 0xf57f54
)

const totalStatusDisplays = 1

func statusLoop(b *Bot) func(*discordgo.Session, *discordgo.Ready) {
	b.logger.Info("ready")
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
					srvCount := b.Bot.Discord.GuildCount()
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

func disconnectHandler(b *Bot) func(*discordgo.Session, *discordgo.Disconnect) {
	return func(s *discordgo.Session, d *discordgo.Disconnect) {
		b.logger.Info("disconnected")
	}
}

func guildBanAddHandler(b *Bot) func(*discordgo.Session, *discordgo.GuildBanAdd) {
	return func(s *discordgo.Session, m *discordgo.GuildBanAdd) {
		g, err := b.Bot.Discord.Guild(m.GuildID)
		if err != nil {
			b.logger.Error("failed to fetch guild", zap.Error(err))
			return
		}

		gc, err := b.db.GetGuild(g.ID)
		if err != nil {
			b.logger.Error("failed to get guild", zap.Error(err))
			return
		}

		embed := builders.NewEmbedBuilder().
			WithTitle("User Banned").
			WithThumbnail(m.User.AvatarURL("256")).
			AddField("User", fmt.Sprintf("%v\n%v", m.User.Mention(), m.User.String()), false).
			AddField("ID", m.User.ID, false).
			WithColor(int(ColorRed))

		if _, err = b.store.GetMember(m.GuildID, m.User.ID); err != nil {
			if err != badger.ErrKeyNotFound {
				b.logger.Error("failed to get member", zap.Error(err))
				return
			}
			embed.WithDescription("User was not in the server")
		}

		// fetch their messages and attachments
		messages, err := b.store.GetMessageLog(m.GuildID, m.User.ID)
		if err != nil {
			b.logger.Error("failed to get message log", zap.Error(err))
			return
		}

		if len(messages) > 0 {
			embed.AddField("Total messages", fmt.Sprint(len(messages)), false)
		}

		reply := builders.NewMessageSendBuilder().Embed(embed.Build())
		_, _ = s.ChannelMessageSendComplex(gc.BanLog, reply.Build())
	}
}

func guildBanRemoveHandler(b *Bot) func(*discordgo.Session, *discordgo.GuildBanRemove) {
	return func(s *discordgo.Session, d *discordgo.GuildBanRemove) {
		g, err := s.State.Guild(d.GuildID)
		if err != nil {
			return
		}

		gc, err := b.db.GetGuild(g.ID)
		if err != nil {
			b.logger.Error("failed to get guild", zap.Error(err))
			return
		}

		embed := builders.NewEmbedBuilder().
			WithTitle("User Unbanned").
			WithThumbnail(d.User.AvatarURL("256")).
			AddField("User", fmt.Sprintf("%v\n%v", d.User.Mention(), d.User.String()), false).
			AddField("ID", d.User.ID, false).
			WithColor(int(ColorGreen))
		_, _ = s.ChannelMessageSendEmbed(gc.UnbanLog, embed.Build())
	}
}

func guildCreateHandler(b *Bot) func(*discordgo.Session, *discordgo.GuildCreate) {
	return func(s *discordgo.Session, d *discordgo.GuildCreate) {

		if _, err := b.db.GetGuild(d.ID); err != nil {
			err = b.db.CreateGuild(d.ID)
			if err != nil {
				b.logger.Error("failed to create new guild", zap.Error(err))
			}
		}

		if len(d.Members) != d.MemberCount {
			_ = s.RequestGuildMembers(d.ID, "", 0, "", false)
			return
		}

		for _, mem := range d.Members {
			err := b.store.SetMember(mem)
			if err != nil {
				b.logger.Error("failed to set member", zap.Error(err))
				continue
			}
		}
	}
}

func guildMemberAddHandler(b *Bot) func(*discordgo.Session, *discordgo.GuildMemberAdd) {
	return func(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
		err := b.store.SetMember(m.Member)
		if err != nil {
			b.logger.Error("failed to set member", zap.Error(err))
		}

		g, err := s.State.Guild(m.GuildID)
		if err != nil {
			return
		}

		gc, err := b.db.GetGuild(g.ID)
		if err != nil {
			b.logger.Error("failed to get guild", zap.Error(err))
			return
		}

		ts := utils.IDToTimestamp(m.User.ID)
		embed := builders.NewEmbedBuilder().
			WithTitle("User Joined").
			WithThumbnail(m.User.AvatarURL("256")).
			AddField("User", fmt.Sprintf("%v\n%v", m.User.Mention(), m.User.String()), false).
			AddField("ID", m.User.ID, false).
			AddField("Creation date", fmt.Sprintf("<t:%v:R>", ts.Unix()), false).
			WithColor(int(ColorBlue))
		_, _ = s.ChannelMessageSendEmbed(gc.JoinLog, embed.Build())
	}
}

func guildMemberRemoveHandler(b *Bot) func(*discordgo.Session, *discordgo.GuildMemberRemove) {
	return func(s *discordgo.Session, m *discordgo.GuildMemberRemove) {
		g, err := s.State.Guild(m.GuildID)
		if err != nil {
			return
		}

		gc, err := b.db.GetGuild(g.ID)
		if err != nil {
			b.logger.Error("failed to get guild", zap.Error(err))
			return
		}

		mem, err := b.store.GetMember(m.GuildID, m.User.ID)
		if err != nil {
			return
		}

		embed := builders.NewEmbedBuilder().
			WithTitle("User Left or Kicked").
			WithThumbnail(m.User.AvatarURL("256")).
			AddField("User", fmt.Sprintf("%v\n%v", m.User.Mention(), m.User.String()), false).
			AddField("ID", m.User.ID, false).
			WithColor(int(ColorOrange))

		var roles []string
		for _, r := range mem.Roles {
			roles = append(roles, fmt.Sprintf("<@&%v>", r))
		}

		if len(roles) <= 0 {
			embed.AddField("Roles", "None", false)
		} else {
			var shown []string
			for _, r := range roles {
				if len(strings.Join(append(shown, r), ", ")) > 760 {
					break
				}
				shown = append(shown, r)
			}

			embedStr := strings.Join(shown, ", ")
			if len(shown) != len(roles) {
				embedStr += fmt.Sprintf(" and %v more", len(roles)-len(shown))
			}
			embed.AddField("Roles", embedStr, false)
		}

		_, _ = s.ChannelMessageSendEmbed(gc.LeaveLog, embed.Build())
		/*
			err = b.store.DeleteMember(m.GuildID, m.User.ID)
			if err != nil {
				b.logger.Error("failed to delete member", zap.Error(err))
			}
		*/
	}
}

func guildMembersChunkHandler(b *Bot) func(*discordgo.Session, *discordgo.GuildMembersChunk) {
	return func(s *discordgo.Session, g *discordgo.GuildMembersChunk) {
		for _, mem := range g.Members {
			err := b.store.SetMember(mem)
			if err != nil {
				b.logger.Error("failed to set member", zap.Error(err))
				continue
			}
		}
	}
}

func guildMemberUpdateHandler(b *Bot) func(*discordgo.Session, *discordgo.GuildMemberUpdate) {
	return func(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {
		err := b.store.SetMember(m.Member)
		if err != nil {
			b.logger.Error("failed to update member", zap.Error(err))
			return
		}
	}
}

func messageCreateHandler(b *Bot) func(*discordgo.Session, *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.Bot {
			return
		}

		// max size 10mb
		_ = b.store.SetMessage(NewDiscordMessage(m.Message, 1024*1024*10))
	}
}

func messageDeleteHandler(b *Bot) func(*discordgo.Session, *discordgo.MessageDelete) {
	return func(s *discordgo.Session, m *discordgo.MessageDelete) {
		msg, err := b.store.GetMessage(m.GuildID, m.ChannelID, m.ID)
		if err != nil {
			return
		}

		g, err := s.State.Guild(m.GuildID)
		if err != nil {
			return
		}

		gc, err := b.db.GetGuild(g.ID)
		if err != nil {
			b.logger.Error("failed to get guild", zap.Error(err))
			return
		}

		embed := builders.NewEmbedBuilder().
			WithTitle("Message Deleted").
			AddField("User", fmt.Sprintf("%v\n%v\n%v", msg.Message.Author.Mention(), msg.Message.Author.String(), msg.Message.Author.ID), true).
			AddField("Message ID", m.ID, true).
			AddField("Channel", fmt.Sprintf("<#%v> (%v)", m.ChannelID, m.ChannelID), false).
			WithDescription("No content").
			WithColor(int(ColorWhite))
		reply := builders.NewMessageSendBuilder()

		if msg.Message.Content != "" {
			str := msg.Message.Content
			if len(str) > 1024 {
				str = "Content too long, so it's put in the attached .txt file"
				reply.AddTextFile("deleted_content.txt", msg.Message.Content)
			}
			embed.WithDescription(str)
		}

		if len(msg.Attachments) > 0 {
			embed.AddField("Total fetched attachments", fmt.Sprint(len(msg.Attachments)), false)
			embed.WithDescription(embed.Description + "\n**Disclaimer:** Only attachments smaller than 10mb may be fetched")
		}

		var files []*discordgo.File
		for _, a := range msg.Attachments {
			files = append(files, &discordgo.File{
				Name:        a.Filename,
				ContentType: "application/octet-stream",
				Reader:      bytes.NewReader(a.Data),
			})
		}
		reply.WithFiles(files).Embed(embed.Build())
		_, _ = s.ChannelMessageSendComplex(gc.MsgDeleteLog, reply.Build())
	}
}

func messageDeleteBulkHandler(b *Bot) func(*discordgo.Session, *discordgo.MessageDeleteBulk) {
	return func(s *discordgo.Session, m *discordgo.MessageDeleteBulk) {
		g, err := b.Bot.Discord.Guild(m.GuildID)
		if err != nil {
			return
		}

		gc, err := b.db.GetGuild(g.ID)
		if err != nil {
			b.logger.Error("failed to get guild", zap.Error(err))
			return
		}

		embed := builders.NewEmbedBuilder().
			WithTitle(fmt.Sprintf("Bulk Message Delete - (%v) messages", len(m.Messages))).
			AddField("Channel", fmt.Sprintf("<#%v>", m.ChannelID), true).
			WithColor(int(ColorWhite))

		var messages []*DiscordMessage
		for _, msgID := range m.Messages {
			msg, err := b.store.GetMessage(m.GuildID, m.ChannelID, msgID)
			if err != nil {
				continue
			}
			messages = append(messages, msg)
		}

		sort.Slice(messages, func(i, j int) bool {
			return messages[i].Message.ID < messages[j].Message.ID
		})

		builder := strings.Builder{}
		builder.WriteString(fmt.Sprintf("%v - %v\n\n\n", m.ChannelID, time.Now().Format(time.RFC3339)))
		for _, msg := range messages {
			text := fmt.Sprintf("\nUser: %v (%v)\nContent: %v\n", msg.Message.Author.String(), msg.Message.Author.ID, msg.Message.Content)
			if len(msg.Attachments) > 0 {
				text += "Message had attachment\n"
			}
			builder.WriteString(text)
		}

		reply := builders.NewMessageSendBuilder().
			AddTextFile(fmt.Sprintf("deleted_%v_%v.txt", m.ChannelID, time.Now().Unix()), builder.String()).
			Embed(embed.Build())

		_, _ = s.ChannelMessageSendComplex(gc.MsgDeleteLog, reply.Build())
	}
}

func messageUpdateHandler(b *Bot) func(*discordgo.Session, *discordgo.MessageUpdate) {
	return func(s *discordgo.Session, m *discordgo.MessageUpdate) {
		// This means it was an image update and not an actual edit
		if m.Message.Content == "" || m.Author.Bot {
			return
		}

		g, err := b.Bot.Discord.Guild(m.GuildID)
		if err != nil {
			b.logger.Info("error", zap.Error(err))
			return
		}

		gc, err := b.db.GetGuild(g.ID)
		if err != nil {
			b.logger.Error("failed to get guild", zap.Error(err))
			return
		}

		oldMsg, err := b.store.GetMessage(m.GuildID, m.ChannelID, m.ID)
		if err != nil || (oldMsg.Message.Author != nil && oldMsg.Message.Author.Bot) {
			return
		}

		if oldMsg.Message.Content == m.Content {
			return
		}

		embed := builders.NewEmbedBuilder().
			WithTitle("Message Edited").
			AddField("User", fmt.Sprintf("%v\n%v\n%v", m.Author.Mention(), m.Author.String(), m.Author.ID), true).
			AddField("Message ID", m.ID, true).
			AddField("Channel", fmt.Sprintf("<#%v> (%v)", m.ChannelID, m.ChannelID), false).
			WithColor(int(ColorBlue))

		reply := builders.NewMessageSendBuilder()

		// check old content
		if len(oldMsg.Message.Content) > 1024 {
			embed.AddField("Old content", "Content too long, so it's put in the attached .txt file", false)
			reply.AddTextFile("old_content.txt", oldMsg.Message.Content)
		} else {
			embed.AddField("Old content", oldMsg.Message.Content, false)
		}

		// check new content
		if len(m.Content) > 1024 {
			embed.AddField("New content", "Content too long, so it's put in the attached .txt file", false)
			reply.AddTextFile("new_content.txt", m.Content)
		} else {
			embed.AddField("New content", m.Content, false)
		}

		reply.Embed(embed.Build())
		_, _ = s.ChannelMessageSendComplex(gc.MsgEditLog, reply.Build())

		// I think this should be put in its own function and not at the end of this one lol
		oldMsg.Message.Content = m.Content
		err = b.store.SetMessage(oldMsg)
		if err != nil {
			b.logger.Error("failed to update message", zap.Error(err))
			return
		}
	}
}
