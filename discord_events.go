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

func disconnectHandler(b *Bot) func(*discordgo.Session, *discordgo.Disconnect) {
	return func(s *discordgo.Session, d *discordgo.Disconnect) {
		b.logger.Info("disconnected")
	}
}

func guildBanAddHandler(b *Bot) func(*discordgo.Session, *discordgo.GuildBanAdd) {
	return func(s *discordgo.Session, d *discordgo.GuildBanAdd) {
		g, err := b.Bot.Discord.Guild(d.GuildID)
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
			WithThumbnail(d.User.AvatarURL("256")).
			AddField("User", fmt.Sprintf("%v\n%v", d.User.Mention(), d.User.String()), false).
			WithFooter(fmt.Sprintf("User ID: %v", d.User.ID), "").
			WithColor(int(ColorRed))

		if _, err = b.store.GetMember(d.GuildID, d.User.ID); err != nil {
			if err != badger.ErrKeyNotFound {
				b.logger.Error("failed to get member", zap.Error(err))
			}
			embed.WithDescription("User was not in the server")
		}

		// fetch their messages and attachments
		messages, err := b.store.GetMessageLog(d.GuildID, d.User.ID)
		if err != nil {
			b.logger.Error("failed to get message log", zap.Error(err))
		}

		builder := strings.Builder{}
		for _, msg := range messages {
			ch, err := b.Bot.Discord.Channel(msg.Message.ChannelID)
			if err != nil {
				b.logger.Error("failed to fetch channel", zap.Error(err))
				continue
			}

			ts := utils.IDToTimestamp(msg.Message.ID).Format(time.DateTime)
			text := fmt.Sprintf("\nChannel: %v (%v)\nTimestamp: %v\nContent: %v\n", ch.Name, ch.ID, ts, msg.Message.Content)
			if len(msg.Attachments) > 0 {
				text += "Info: Message had attachment\n"
			}
			builder.WriteString(text)
		}

		if len(messages) > 0 {
			embed.WithDescription(embed.Description + "\n24 hour message log is attached")
			embed.AddField("Total messages", fmt.Sprint(len(messages)), false)
		}

		reply := builders.NewMessageSendBuilder().
			AddTextFile(fmt.Sprintf("24h_ban_log_%v_%v.txt", d.User.ID, time.Now().Unix()), builder.String()).
			Embed(embed.Build())
		_, _ = s.ChannelMessageSendComplex(gc.BanLog, reply.Build())
	}
}

func guildBanRemoveHandler(b *Bot) func(*discordgo.Session, *discordgo.GuildBanRemove) {
	return func(s *discordgo.Session, d *discordgo.GuildBanRemove) {
		g, err := b.Bot.Discord.Guild(d.GuildID)
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
			WithFooter(fmt.Sprintf("User ID: %v", d.User.ID), "").
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
	return func(s *discordgo.Session, d *discordgo.GuildMemberAdd) {
		err := b.store.SetMember(d.Member)
		if err != nil {
			b.logger.Error("failed to set member", zap.Error(err))
		}

		g, err := b.Bot.Discord.Guild(d.GuildID)
		if err != nil {
			return
		}

		gc, err := b.db.GetGuild(g.ID)
		if err != nil {
			b.logger.Error("failed to get guild", zap.Error(err))
			return
		}

		ts := utils.IDToTimestamp(d.User.ID)
		embed := builders.NewEmbedBuilder().
			WithTitle("User Joined").
			WithThumbnail(d.User.AvatarURL("256")).
			AddField("User", fmt.Sprintf("%v\n%v", d.User.Mention(), d.User.String()), false).
			AddField("Creation date", fmt.Sprintf("<t:%v:R>", ts.Unix()), false).
			WithFooter(fmt.Sprintf("User ID: %v", d.User.ID), "").
			WithColor(int(ColorBlue))
		_, _ = s.ChannelMessageSendEmbed(gc.JoinLog, embed.Build())
	}
}

func guildMemberRemoveHandler(b *Bot) func(*discordgo.Session, *discordgo.GuildMemberRemove) {
	return func(s *discordgo.Session, d *discordgo.GuildMemberRemove) {
		g, err := b.Bot.Discord.Guild(d.GuildID)
		if err != nil {
			return
		}

		gc, err := b.db.GetGuild(g.ID)
		if err != nil {
			b.logger.Error("failed to get guild", zap.Error(err))
			return
		}

		mem, err := b.store.GetMember(d.GuildID, d.User.ID)
		if err != nil {
			return
		}

		embed := builders.NewEmbedBuilder().
			WithTitle("User Left or Kicked").
			WithThumbnail(d.User.AvatarURL("256")).
			AddField("User", fmt.Sprintf("%v\n%v", d.User.Mention(), d.User.String()), false).
			WithFooter(fmt.Sprintf("User ID: %v", d.User.ID), "").
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
		err = b.store.DeleteMember(d.GuildID, d.User.ID)
		if err != nil {
			b.logger.Error("failed to delete member", zap.Error(err))
		}
	}
}

func guildMembersChunkHandler(b *Bot) func(*discordgo.Session, *discordgo.GuildMembersChunk) {
	return func(s *discordgo.Session, d *discordgo.GuildMembersChunk) {
		for _, mem := range d.Members {
			err := b.store.SetMember(mem)
			if err != nil {
				b.logger.Error("failed to set member", zap.Error(err))
				continue
			}
		}
	}
}

func guildMemberUpdateHandler(b *Bot) func(*discordgo.Session, *discordgo.GuildMemberUpdate) {
	return func(s *discordgo.Session, d *discordgo.GuildMemberUpdate) {
		err := b.store.SetMember(d.Member)
		if err != nil {
			b.logger.Error("failed to update member", zap.Error(err))
			return
		}
	}
}

func messageCreateHandler(b *Bot) func(*discordgo.Session, *discordgo.MessageCreate) {
	return func(s *discordgo.Session, d *discordgo.MessageCreate) {
		if d.Author.Bot {
			return
		}

		// max size 10mb
		_ = b.store.SetMessage(NewDiscordMessage(d.Message, 1024*1024*10))
	}
}

func messageDeleteHandler(b *Bot) func(*discordgo.Session, *discordgo.MessageDelete) {
	return func(s *discordgo.Session, d *discordgo.MessageDelete) {
		msg, err := b.store.GetMessage(d.GuildID, d.ChannelID, d.ID)
		if err != nil {
			return
		}

		g, err := b.Bot.Discord.Guild(d.GuildID)
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
			AddField("Channel", fmt.Sprintf("<#%v> (%v)", d.ChannelID, d.ChannelID), false).
			WithFooter(fmt.Sprintf("Message ID: %v", d.ID), "").
			WithColor(int(ColorWhite))
		reply := builders.NewMessageSendBuilder()

		descStr := "No content"
		if msg.Message.Content != "" {
			descStr = msg.Message.Content
			if len(descStr) > 1024 {
				descStr = "Content too long, so it's put in the attached .txt file"
				reply.AddTextFile("deleted_content.txt", msg.Message.Content)
			}
			embed.WithDescription(descStr)
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
	return func(s *discordgo.Session, d *discordgo.MessageDeleteBulk) {
		g, err := b.Bot.Discord.Guild(d.GuildID)
		if err != nil {
			return
		}

		gc, err := b.db.GetGuild(g.ID)
		if err != nil {
			b.logger.Error("failed to get guild", zap.Error(err))
			return
		}

		embed := builders.NewEmbedBuilder().
			WithTitle(fmt.Sprintf("Bulk Message Delete - (%v) messages", len(d.Messages))).
			AddField("Channel", fmt.Sprintf("<#%v>", d.ChannelID), true).
			WithColor(int(ColorWhite))

		var messages []*DiscordMessage
		for _, msgID := range d.Messages {
			msg, err := b.store.GetMessage(d.GuildID, d.ChannelID, msgID)
			if err != nil {
				continue
			}
			messages = append(messages, msg)
		}

		sort.Slice(messages, func(i, j int) bool {
			return messages[i].Message.ID < messages[j].Message.ID
		})

		builder := strings.Builder{}
		for _, msg := range messages {
			text := fmt.Sprintf("\nUser: %v (%v)\nContent: %v\n", msg.Message.Author.String(), msg.Message.Author.ID, msg.Message.Content)
			if len(msg.Attachments) > 0 {
				text += "Message had attachment\n"
			}
			builder.WriteString(text)
		}

		reply := builders.NewMessageSendBuilder().
			AddTextFile(fmt.Sprintf("deleted_%v_%v.txt", d.ChannelID, time.Now().Unix()), builder.String()).
			Embed(embed.Build())

		_, _ = s.ChannelMessageSendComplex(gc.MsgDeleteLog, reply.Build())
	}
}

func messageUpdateHandler(b *Bot) func(*discordgo.Session, *discordgo.MessageUpdate) {
	return func(s *discordgo.Session, d *discordgo.MessageUpdate) {
		// This means it was an image update and not an actual edit
		if d.Message.Content == "" || d.Author.Bot {
			return
		}

		g, err := b.Bot.Discord.Guild(d.GuildID)
		if err != nil {
			b.logger.Info("error", zap.Error(err))
			return
		}

		gc, err := b.db.GetGuild(g.ID)
		if err != nil {
			b.logger.Error("failed to get guild", zap.Error(err))
			return
		}

		oldMsg, err := b.store.GetMessage(d.GuildID, d.ChannelID, d.ID)
		if err != nil || (oldMsg.Message.Author != nil && oldMsg.Message.Author.Bot) {
			return
		}

		if oldMsg.Message.Content == d.Content {
			return
		}

		embed := builders.NewEmbedBuilder().
			WithTitle("Message Edited").
			AddField("User", fmt.Sprintf("%v\n%v\n%v", d.Author.Mention(), d.Author.String(), d.Author.ID), true).
			AddField("Channel", fmt.Sprintf("<#%v> (%v)", d.ChannelID, d.ChannelID), false).
			WithFooter(fmt.Sprintf("Message ID: %v", d.ID), "").
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
		if len(d.Content) > 1024 {
			embed.AddField("New content", "Content too long, so it's put in the attached .txt file", false)
			reply.AddTextFile("new_content.txt", d.Content)
		} else {
			embed.AddField("New content", d.Content, false)
		}

		reply.Embed(embed.Build())
		_, _ = s.ChannelMessageSendComplex(gc.MsgEditLog, reply.Build())

		// I think this should be put in its own function and not at the end of this one lol
		oldMsg.Message.Content = d.Content
		err = b.store.SetMessage(oldMsg)
		if err != nil {
			b.logger.Error("failed to update message", zap.Error(err))
			return
		}
	}
}
