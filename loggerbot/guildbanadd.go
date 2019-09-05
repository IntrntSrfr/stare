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

func (b *Bot) guildBanAddHandler(s *discordgo.Session, m *discordgo.GuildBanAdd) {

	dg := Guild{}
	b.db.Get(&dg, "SELECT ban_log FROM guilds WHERE id=$1;", m.GuildID)
	if dg.BanLog == "" {
		return
	}

	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		b.logger.Info("error", zap.Error(err))
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

	_, err = b.loggerDB.GetMember(fmt.Sprintf("%v:%v", m.GuildID, m.User.ID))
	if err != nil {
		embed.Title += " - Hackban"

		_, err = s.ChannelMessageSendEmbed(dg.BanLog, &embed)
		if err != nil {
			b.logger.Info("error", zap.Error(err))
			fmt.Println("BAN LOG ERROR", err)
		}
	} else {

		messagelog, err := b.loggerDB.GetMessageLog(m)
		if err != nil {
			fmt.Println(err)
			return
		}

		text := strings.Builder{}
		sort.Sort(loggerdb.ByID(messagelog))

		for _, cmsg := range messagelog {
			if cmsg.Message.Author.ID == m.User.ID {

				ch, err := s.State.Channel(cmsg.Message.ChannelID)
				if err != nil {
					continue
				}

				if len(cmsg.Attachments) > 0 {
					text.WriteString(fmt.Sprintf("\nUser: %v (%v)\nChannel: %v (%v)\nContent: %v\nMessage had attachment\n", cmsg.Message.Author.String(), cmsg.Message.Author.ID, ch.Name, ch.ID, cmsg.Message.Content))
				} else {
					text.WriteString(fmt.Sprintf("\nUser: %v (%v)\nChannel: %v (%v)\nContent: %v\n", cmsg.Message.Author.String(), cmsg.Message.Author.ID, ch.Name, ch.ID, cmsg.Message.Content))
				}
			}
		}

		if len(messagelog) > 0 {
			if b.config.OwoAPIKey != "" {
				link, err := b.owo.Upload(text.String())
				if err != nil {
					embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
						Name:  "24h user log",
						Value: "Error getting link",
					})
					b.logger.Info("BAN LOG ERROR", zap.Error(err))
				} else {
					embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
						Name:  "24h user log",
						Value: "Link: " + link,
					})
				}

				_, err = s.ChannelMessageSendEmbed(dg.BanLog, &embed)
				if err != nil {
					fmt.Println("BAN LOG ERROR", err)
				}
			} else {
				jeff := bytes.Buffer{}
				jeff.WriteString(text.String())

				msg, err := s.ChannelMessageSendEmbed(dg.BanLog, &embed)
				if err != nil {
					fmt.Println("BAN LOG ERROR", err)
				}

				s.ChannelFileSendWithMessage(dg.BanLog, fmt.Sprintf("Log file for delete log message ID %v:", msg.ID), "banlog_"+m.User.ID+".txt", &jeff)
			}
		} else {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  "24h user log",
				Value: "No history.",
			})

			_, err = s.ChannelMessageSendEmbed(dg.BanLog, &embed)
			if err != nil {
				b.logger.Info("error", zap.Error(err))
				fmt.Println("BAN LOG ERROR", err)
			}
		}
	}
}
