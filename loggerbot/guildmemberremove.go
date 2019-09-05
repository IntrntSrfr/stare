package loggerbot

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

func (b *Bot) guildMemberRemoveHandler(s *discordgo.Session, m *discordgo.GuildMemberRemove) {

	dg := Guild{}
	b.db.Get(&dg, "SELECT leave_log FROM guilds WHERE id=$1;", m.GuildID)
	if dg.LeaveLog == "" {
		return
	}

	roles := []string{}

	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		b.logger.Info("error", zap.Error(err))
		return
	}

	mem, err := b.loggerDB.GetMember(fmt.Sprintf("%v:%v", m.GuildID, m.User.ID))
	if err != nil {
		b.logger.Info("error", zap.Error(err))
		return
	}

	for _, r := range mem.Roles {
		roles = append(roles, fmt.Sprintf("<@&%v>", r))
	}

	embed := discordgo.MessageEmbed{
		Color: dColorOrange,
		Title: "User left or kicked",
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: m.User.AvatarURL("256"),
		},
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
				Name:   "User",
				Value:  fmt.Sprintf("%v\n%v", m.User.Mention(), m.User.String()),
				Inline: true,
			},
			&discordgo.MessageEmbedField{
				Name:   "ID",
				Value:  m.User.ID,
				Inline: true,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			IconURL: discordgo.EndpointGuildIcon(g.ID, g.Icon),
			Text:    g.Name,
		},
	}

	if len(roles) < 1 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Roles",
			Value: "None",
		})
	} else if len(roles) < 10 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Roles",
			Value: strings.Join(roles, ", "),
		})
	} else {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Roles",
			Value: fmt.Sprintf("%v and %v more", strings.Join(roles[0:9], ", "), len(roles)-9),
		})
	}

	_, err = s.ChannelMessageSendEmbed(dg.LeaveLog, &embed)
	if err != nil {
		b.logger.Info("error", zap.Error(err))
		fmt.Println("LEAVE LOG ERROR", err)
	}

	err = b.loggerDB.DeleteMember(fmt.Sprintf("%v:%v", m.GuildID, m.User.ID))
	if err != nil {
		b.logger.Info("error", zap.Error(err))
		fmt.Println(err)
		return
	}
}
