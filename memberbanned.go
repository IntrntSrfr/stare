package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ninedraft/simplepaste"
)

func MemberBannedHandler(s *discordgo.Session, m *discordgo.GuildBanAdd) {

	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		return
	}

	ts := time.Now()

	embed := discordgo.MessageEmbed{
		Color: dColorRed,
		Title: "User banned",
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: m.User.AvatarURL("256"),
		},
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
				Name:  "User",
				Value: fmt.Sprintf("%v\n%v", m.User.Mention(), m.User.String()),
			},
			&discordgo.MessageEmbedField{
				Name:  "ID",
				Value: m.User.ID,
			},
		},
		Timestamp: ts.Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			IconURL: discordgo.EndpointGuildIcon(g.ID, g.Icon),
			Text:    g.Name,
		},
	}

	if _, ok := memCache.Get(fmt.Sprint("%v:%v", m.GuildID, m.User.ID)); ok {

		text := ""
		msgCount := 0

		bmsgs := []*DiscMessage{}
		for _, cmsg := range msgCache.storage {
			if cmsg.Message.Author.ID == m.User.ID {
				bmsgs = append(bmsgs, cmsg)
			}
		}

		sort.Sort(ByID(bmsgs))

		for _, cmsg := range bmsgs {
			if cmsg.Message.Author.ID == m.User.ID {

				ch, err := s.State.Channel(cmsg.Message.ChannelID)
				if err != nil {
					continue
				}

				if len(cmsg.Attachment) > 0 {
					text += fmt.Sprintf("\nUser: %v (%v)\nChannel: %v (%v)\nContent: %v\nMessage had attachment\n", cmsg.Message.Author.String(), cmsg.Message.Author.ID, ch.Name, ch.ID, cmsg.Message.Content)
				} else {
					text += fmt.Sprintf("\nUser: %v (%v)\nChannel: %v (%v)\nContent: %v\n", cmsg.Message.Author.String(), cmsg.Message.Author.ID, ch.Name, ch.ID, cmsg.Message.Content)
				}
				msgCount++
			}
		}

		if msgCount > 0 {

			paste := simplepaste.NewPaste(fmt.Sprintf("24h ban log for %v (%v) - %v", m.User.String(), m.User.ID, ts.Format(time.RFC1123)), text)

			paste.ExpireDate = simplepaste.Never
			paste.Privacy = simplepaste.Unlisted

			link, err := api.SendPaste(paste)
			if err != nil {
				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
					Name:  "24h user log",
					Value: "Error getting pastebin link",
				})
			} else {
				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
					Name:  "24h user log",
					Value: link,
				})
			}
		} else {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  "24h user log",
				Value: "No history.",
			})
		}
	} else {
		embed.Title += " - Hackban"
	}

	_, err = s.ChannelMessageSendEmbed(config.Ban, &embed)
	if err != nil {
		fmt.Println("BAN LOG ERROR", err)
	}
}
