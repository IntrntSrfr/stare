package loggerbot

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

func (b *Bot) guildBanRemoveHandler(s *discordgo.Session, m *discordgo.GuildBanRemove) {

	dg := Guild{}
	b.db.Get(&dg, "SELECT unban_log FROM guilds WHERE id=$1;", m.GuildID)
	if dg.UnbanLog == "" {
		return
	}

	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		b.logger.Info("error", zap.Error(err))
		return
	}

	embed := discordgo.MessageEmbed{
		Color: dColorGreen,
		Title: "User unbanned",
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
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			IconURL: discordgo.EndpointGuildIcon(g.ID, g.Icon),
			Text:    g.Name,
		},
	}
	_, err = s.ChannelMessageSendEmbed(dg.UnbanLog, &embed)
	if err != nil {
		b.logger.Info("error", zap.Error(err))
		fmt.Println("UNBAN LOG ERROR", err)
	}
}
