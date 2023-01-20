package main

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

func MessageUpdateHandler(s *discordgo.Session, m *discordgo.MessageUpdate) {

	if oldm, ok := msgCache.Get(m.ID); ok {

		g, err := s.State.Guild(m.GuildID)
		if err != nil {
			return
		}

		oldmsg := oldm.Message

		if oldmsg.Content == m.Content {
			return
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
					Value: oldmsg.Content,
				},
				&discordgo.MessageEmbedField{
					Name:  "New content",
					Value: m.Content,
				},
			},
			Timestamp: time.Now().Format(time.RFC3339),
			Footer: &discordgo.MessageEmbedFooter{
				IconURL: discordgo.EndpointGuildIcon(g.ID, g.Icon),
				Text:    g.Name,
			},
		}
		_, err = s.ChannelMessageSendEmbed(config.MsgEdit, &embed)
		if err != nil {
			fmt.Println("EDIT LOG ERROR", err)
		}

		go msgCache.Update(&DiscMessage{
			Attachment: oldm.Attachment,
			Message:    m.Message,
		})
	}
}
