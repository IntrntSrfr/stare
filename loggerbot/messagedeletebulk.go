package loggerbot

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/intrntsrfr/functional-logger/loggerdb"
	"go.uber.org/zap"
)

func (b *Bot) messageDeleteBulkHandler(s *discordgo.Session, m *discordgo.MessageDeleteBulk) {

	dg := Guild{}
	b.db.Get(&dg, "SELECT msg_delete_log FROM guilds WHERE id=$1;", m.GuildID)
	if dg.MsgDeleteLog == "" {
		return
	}

	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		b.logger.Info("error", zap.Error(err))
		return
	}
	ts := time.Now()

	embed := discordgo.MessageEmbed{
		Color: dColorWhite,
		Title: fmt.Sprintf("Bulk message delete - (%v) messages deleted", len(m.Messages)),
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
				Name:   "Channel",
				Value:  fmt.Sprintf("<#%v>", m.ChannelID),
				Inline: true,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			IconURL: discordgo.EndpointGuildIcon(g.ID, g.Icon),
			Text:    g.Name,
		},
	}
	deletedmsgs := []*loggerdb.DMsg{}
	for _, msgid := range m.Messages {
		delmsg, err := b.loggerDB.GetMessage(fmt.Sprintf("%v:%v:%v", m.GuildID, m.ChannelID, msgid))
		if err != nil {
			b.logger.Info("error", zap.Error(err))
			continue
		}
		deletedmsgs = append(deletedmsgs, delmsg)
	}

	sort.Sort(loggerdb.ByID(deletedmsgs))

	text := strings.Builder{}
	text.WriteString(fmt.Sprintf("%v - %v\n\n\n", m.ChannelID, ts.Format(time.RFC1123)))

	for _, msg := range deletedmsgs {
		if len(msg.Attachments) > 0 {
			text.WriteString(fmt.Sprintf("\nUser: %v (%v)\nContent: %v\nMessage had attachment\n", msg.Message.Author.String(), msg.Message.Author.ID, msg.Message.Content))
		} else {
			text.WriteString(fmt.Sprintf("\nUser: %v (%v)\nContent: %v\n", msg.Message.Author.String(), msg.Message.Author.ID, msg.Message.Content))
		}
	}

	if b.config.OwoAPIKey != "" {
		res, err := b.owo.Upload(text.String())

		if err != nil {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  "Logged messages:",
				Value: "Error getting link",
			})
			b.logger.Info("BULK DELETE LOG ERROR", zap.Error(err))
		} else {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  "Logged messages:",
				Value: "Link: " + res,
			})
		}
		_, err = s.ChannelMessageSendEmbed(dg.MsgDeleteLog, &embed)
		if err != nil {
			fmt.Println("BULK DELETE LOG ERROR", err)
		}
	} else {
		jeff := bytes.Buffer{}
		jeff.WriteString(text.String())

		msg, err := s.ChannelMessageSendEmbed(dg.MsgDeleteLog, &embed)
		if err != nil {
			fmt.Println("BULK DELETE LOG ERROR", err)
		}

		s.ChannelFileSendWithMessage(dg.MsgDeleteLog, fmt.Sprintf("Log file for delete log message ID %v:", msg.ID), "deletelog_"+m.ChannelID+".txt", &jeff)
	}
}
