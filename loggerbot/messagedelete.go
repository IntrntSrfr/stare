package loggerbot

import (
	"bytes"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

func (b *Bot) messageDeleteHandler(s *discordgo.Session, m *discordgo.MessageDelete) {

	dg := Guild{}
	b.db.Get(&dg, "SELECT msg_delete_log FROM guilds WHERE id=$1;", m.GuildID)
	if dg.MsgDeleteLog == "" {
		return
	}

	msg, err := b.loggerDB.GetMessage(fmt.Sprintf("%v:%v:%v", m.GuildID, m.ChannelID, m.ID))
	if err != nil {
		//fmt.Println(err)
		return
	}
	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		b.logger.Info("error", zap.Error(err))
		return
	}

	embed := discordgo.MessageEmbed{
		Color: dColorWhite,
		Title: "Message deleted",
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
				Name:   "User",
				Value:  fmt.Sprintf("%v\n%v\n%v", msg.Message.Author.Mention(), msg.Message.Author.String(), msg.Message.Author.ID),
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
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			IconURL: discordgo.EndpointGuildIcon(g.ID, g.Icon),
			Text:    g.Name,
		},
	}

	if msg.Message.Content != "" {
		content := ""
		if len(msg.Message.Content) > 1024 {
			link, err := b.owo.Upload(msg.Message.Content)
			if err != nil {
				content = "Content unavailable"
			} else {
				content = "Message too big for embed, have a link instead: " + link
			}
		} else {
			content = msg.Message.Content
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Content",
			Value: content,
		})
	} else {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Content",
			Value: "No content",
		})
	}

	if len(msg.Message.Attachments) > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Total attachments",
			Value: fmt.Sprint(len(msg.Message.Attachments)),
		})
	}

	_, err = s.ChannelMessageSendEmbed(dg.MsgDeleteLog, &embed)
	if err != nil {
		b.logger.Info("error", zap.Error(err))
		fmt.Println("DELETE LOG ERROR", err)
	}
	if len(msg.Message.Attachments) > 0 {
		send, err := s.ChannelMessageSend(dg.MsgDeleteLog, "Trying to get attachments..")
		if err != nil {
			b.logger.Info("error", zap.Error(err))
			fmt.Println("DELETE LOG SEND ERROR", err)
			return
		}

		data := &discordgo.MessageSend{
			Content: fmt.Sprintf("File(s) attached to message ID: %v", m.ID),
		}

		for k, img := range msg.Attachments {
			f := &discordgo.File{
				Name:   msg.Message.Attachments[k].Filename,
				Reader: bytes.NewReader(img),
			}
			data.Files = append(data.Files, f)
		}

		_, err = s.ChannelMessageSendComplex(dg.MsgDeleteLog, data)
		if err != nil {
			b.logger.Info("error", zap.Error(err))
			s.ChannelMessageEdit(send.ChannelID, send.ID, "Error getting attachments")
			fmt.Println("DELETE LOG ERROR", err)
		} else {
			s.ChannelMessageDelete(send.ChannelID, send.ID)
		}
	}
}
