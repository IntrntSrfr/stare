package loggerbot

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

func (b *Bot) messageUpdateHandler(s *discordgo.Session, m *discordgo.MessageUpdate) {

	row := b.db.QueryRow("SELECT msgeditlog FROM discordguilds WHERE guildid=$1;", m.GuildID)
	dg := DiscordGuild{}
	err := row.Scan(&dg.MsgEditLog)
	if err != nil {
		fmt.Println(err)
		return
	}
	if dg.MsgEditLog == "" {
		return
	}

	// This means it was an image update and not an actual edit
	if m.Message.Content == "" {
		return
	}

	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		b.logger.Info("error", zap.Error(err))
		return
	}

	oldm, err := b.loggerDB.GetMessage(fmt.Sprintf("%v:%v:%v", m.GuildID, m.ChannelID, m.ID))
	if err != nil {
		return
	}

	oldmsg := oldm.Message

	if oldmsg.Content == m.Content {
		return
	}

	oldc := ""
	newc := ""

	if len(m.Content) > 1024 {
		link, err := b.owo.Upload(m.Content)
		if err != nil {
			newc = "Content unavailable"
		} else {
			newc = "Message too big for embed, have a link instead: " + link
		}
	} else {
		newc = m.Content
	}

	if len(oldmsg.Content) > 1024 {
		link, err := b.owo.Upload(oldmsg.Content)
		if err != nil {
			oldc = "Content unavailable"
		} else {
			oldc = "Message too big for embed, have a link instead: " + link
		}
	} else {
		oldc = oldmsg.Content
	}

	embed := discordgo.MessageEmbed{
		Color: dColorLBlue,
		Title: "Message edited",
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
				Name:   "User",
				Value:  fmt.Sprintf("%v\n%v\n%v", oldmsg.Author.Mention(), oldmsg.Author.String(), oldmsg.Author.ID),
				Inline: true,
			},
			&discordgo.MessageEmbedField{
				Name:   "Message ID",
				Value:  m.ID,
				Inline: true,
			},
			&discordgo.MessageEmbedField{
				Name:  "Channel",
				Value: fmt.Sprintf("<#%v> (%v)", m.ChannelID, m.ChannelID),
			},
			&discordgo.MessageEmbedField{
				Name:  "Old content",
				Value: oldc,
			},
			&discordgo.MessageEmbedField{
				Name:  "New content",
				Value: newc,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			IconURL: discordgo.EndpointGuildIcon(g.ID, g.Icon),
			Text:    g.Name,
		},
	}

	_, err = s.ChannelMessageSendEmbed(dg.MsgEditLog, &embed)
	if err != nil {
		b.logger.Info("error", zap.Error(err))
		fmt.Println("EDIT LOG ERROR", err)
	}

	oldm.Message.Content = m.Content

	err = b.loggerDB.SetMessage(oldm.Message)
	if err != nil {
		b.logger.Info("error", zap.Error(err))
		fmt.Println("ERROR")
		return
	}
}
