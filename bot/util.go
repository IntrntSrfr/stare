package bot

import (
	"bytes"
	"github.com/bwmarrin/discordgo"
	"strconv"
	"strings"
	"time"
)

func TrimChannelString(chStr string) string {
	chStr = strings.TrimPrefix(chStr, "<#")
	chStr = strings.TrimSuffix(chStr, ">")
	return chStr
}

func ParseSnowflake(id string) (time.Time, error) {
	n, err := strconv.ParseInt(id, 0, 63)
	if err != nil {
		return time.Now(), err
	}
	return time.Unix(((n>>22)+1420070400000)/1000, 0), nil
}

func NewLogEmbed(t LogType, g *discordgo.Guild) *discordgo.MessageEmbed {
	e := &discordgo.MessageEmbed{
		Fields:    []*discordgo.MessageEmbedField{},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if g != nil {
		e.Footer = &discordgo.MessageEmbedFooter{
			IconURL: discordgo.EndpointGuildIcon(g.ID, g.Icon),
			Text:    g.Name,
		}
	}

	switch t {
	case MessageDeleteType:
		e.Title = "Message deleted"
		e.Color = White
	case MessageDeleteBulkType:
		e.Title = "Multiple messages deleted"
		e.Color = White
	case MessageUpdateType:
		e.Title = "Message edited"
		e.Color = Blue
	case GuildJoinType:
		e.Title = "User joined"
		e.Color = Blue
	case GuildLeaveType:
		e.Title = "User left or kicked"
		e.Color = Orange
	case GuildBanType:
		e.Title = "User banned"
		e.Color = Red
	case GuildUnbanType:
		e.Title = "User unbanned"
		e.Color = Green
	}
	return e
}

func AddMessageFile(m *discordgo.MessageSend, filename string, data []byte) *discordgo.MessageSend {
	m.Files = append(m.Files, &discordgo.File{
		Name:   filename,
		Reader: bytes.NewBuffer(data),
	})
	return m
}

func AddMessageFileString(m *discordgo.MessageSend, filename, data string) *discordgo.MessageSend {
	return AddMessageFile(m, filename, []byte(data))
}

func AddEmbedField(e *discordgo.MessageEmbed, name, value string, inline bool) *discordgo.MessageEmbed {
	e.Fields = append(e.Fields, &discordgo.MessageEmbedField{
		Name:   name,
		Value:  value,
		Inline: inline,
	})
	return e
}

func SetEmbedThumbnail(e *discordgo.MessageEmbed, url string) *discordgo.MessageEmbed {
	e.Thumbnail = &discordgo.MessageEmbedThumbnail{
		URL: url,
	}
	return e
}
