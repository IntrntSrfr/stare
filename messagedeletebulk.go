package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ninedraft/simplepaste"
)

func MessageDeleteBulkHandler(s *discordgo.Session, m *discordgo.MessageDeleteBulk) {

	g, err := s.State.Guild(m.GuildID)
	if err != nil {
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

	deletedmsgs := []*DiscMessage{}
	for _, mc := range m.Messages {
		if msg, ok := msgCache.Get(mc); ok {
			deletedmsgs = append(deletedmsgs, msg)
		}
	}

	sort.Sort(ByID(deletedmsgs))

	text := ""

	for _, msg := range deletedmsgs {
		if len(msg.Attachment) > 0 {
			text += fmt.Sprintf("\nUser: %v (%v)\nContent: %v\nMessage had attachment\n", msg.Message.Author.String(), msg.Message.Author.ID, msg.Message.Content)
		} else {
			text += fmt.Sprintf("\nUser: %v (%v)\nContent: %v\n", msg.Message.Author.String(), msg.Message.Author.ID, msg.Message.Content)
		}
	}

	paste := simplepaste.NewPaste(fmt.Sprintf("%v - %v", m.ChannelID, ts.Format(time.RFC1123)), text)

	paste.ExpireDate = simplepaste.Never
	paste.Privacy = simplepaste.Unlisted

	link, err := api.SendPaste(paste)
	if err != nil {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Pastebin log link",
			Value: "Error getting pastebin link",
		})
	} else {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Pastebin log link",
			Value: link,
		})
	}

	_, err = s.ChannelMessageSendEmbed(config.MsgDelete, &embed)
	if err != nil {
		fmt.Println("BULK DELETE LOG ERROR", err)
	}
}
