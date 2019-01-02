package main

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
)

func MemberJoinedHandler(s *discordgo.Session, m *discordgo.GuildMemberAdd) {

	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		return
	}

	if _, ok := memCache.Get(fmt.Sprint("%v:%v", m.GuildID, m.User.ID)); !ok {
		go memCache.Put(m.Member)
	}

	id, err := strconv.ParseInt(m.User.ID, 0, 63)
	if err != nil {
		return
	}

	id = ((id >> 22) + 1420070400000) / 1000

	dur := time.Since(time.Unix(int64(id), 0))

	ts := time.Unix(id, 0)

	embed := discordgo.MessageEmbed{
		Color: dColorLBlue,
		Title: "User joined",
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: m.User.AvatarURL("256"),
		},
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
				Name:  "User",
				Value: fmt.Sprintf("%v\n%v (%v)", m.User.Mention(), m.User.String(), m.User.ID),
			},
			&discordgo.MessageEmbedField{
				Name:  "Creation date",
				Value: fmt.Sprintf("%v\n%v days ago", ts.Format(time.RFC1123), math.Floor(dur.Hours()/float64(24))),
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			IconURL: discordgo.EndpointGuildIcon(g.ID, g.Icon),
			Text:    g.Name,
		},
	}

	_, err = s.ChannelMessageSendEmbed(config.Join, &embed)
	if err != nil {
		fmt.Println("JOIN LOG ERROR", err)
	}
}
