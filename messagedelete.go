package main

import (
	"bytes"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

func MessageDeleteHandler(s *discordgo.Session, m *discordgo.MessageDelete) {

	if msg, ok := msgCache.Get(m.ID); ok {
		g, err := s.State.Guild(m.GuildID)
		if err != nil {
			return
		}

		msgo := msg.Message

		embed := discordgo.MessageEmbed{
			Color: dColorWhite,
			Title: "Message deleted",
			Fields: []*discordgo.MessageEmbedField{
				&discordgo.MessageEmbedField{
					Name:   "User",
					Value:  fmt.Sprintf("%v\n%v\n%v", msgo.Author.Mention(), msgo.Author.String(), msgo.Author.ID),
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

		if msgo.Content != "" {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  "Content",
				Value: msgo.Content,
			})
		} else {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  "Content",
				Value: "No content",
			})
		}
		if len(msgo.Attachments) > 0 {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  "Total attachments",
				Value: fmt.Sprint(len(msgo.Attachments)),
			})
		}

		_, err = s.ChannelMessageSendEmbed(config.MsgDelete, &embed)
		if err != nil {
			fmt.Println("DELETE LOG ERROR", err)
		}
		if len(msgo.Attachments) > 0 {
			send, err := s.ChannelMessageSend(config.MsgDelete, "Trying to get attachments..")
			if err != nil {
				fmt.Println("DELETE LOG SEND ERROR", err)
				return
			}
			data := &discordgo.MessageSend{
				Content: fmt.Sprintf("File(s) attached to message ID:%v", m.ID),
			}

			for k, img := range msg.Attachment {
				f := &discordgo.File{
					Name:   msgo.Attachments[k].Filename,
					Reader: bytes.NewReader(img),
				}
				data.Files = append(data.Files, f)
			}

			_, err = s.ChannelMessageSendComplex(config.MsgDelete, data)
			if err != nil {
				s.ChannelMessageEdit(send.ChannelID, send.ID, "Error getting attachments")
				fmt.Println("DELETE LOG ERROR", err)
			} else {
				s.ChannelMessageDelete(send.ChannelID, send.ID)
			}
		}
	}
}
