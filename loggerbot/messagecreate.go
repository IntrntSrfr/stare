package loggerbot

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

func (b *Bot) messageCreateHandler(s *discordgo.Session, m *discordgo.MessageCreate) {

	if m.Author.Bot {
		return
	}

	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		b.logger.Info("error", zap.Error(err))
		fmt.Println(err)
		return
	}

	ch, err := s.State.Channel(m.ChannelID)
	if err != nil {
		b.logger.Info("error", zap.Error(err))
		fmt.Println(err)
		return
	}
	if ch.Type != discordgo.ChannelTypeGuildText {
		return
	}

	fmt.Println(fmt.Sprintf("%v - %v - %v: %v", g.Name, ch.Name, m.Author.String(), m.Content))

	go b.loggerDB.SetMessage(m.Message)
	/*
		err = loggerDB.SetMessage(m.Message)
		if err != nil {
			fmt.Println("MESSAGE CREATE ERROR", err)
			return
		}
	*/
	if strings.HasPrefix(m.Content, "fl.len") {
		s.ChannelMessageSend(ch.ID, fmt.Sprintf("messages: %v", b.loggerDB.TotalMessages))
	} else if strings.HasPrefix(m.Content, "fl.mlen") {
		s.ChannelMessageSend(ch.ID, fmt.Sprintf("members: %v", b.loggerDB.TotalMembers))
	} else if strings.HasPrefix(m.Content, "fl.uptime") {
		s.ChannelMessageSend(ch.ID, fmt.Sprintf("%v", fmt.Sprintf("Uptime: %v", time.Now().Sub(b.starttime).Round(time.Second).String())))
	}

	args := strings.Split(m.Content, " ")

	if args[0] == "fl.set" {
		if len(args) < 2 {
			return
		}
		channel := ch
		chstr := ""
		if len(args) > 2 {
			if strings.HasPrefix(args[2], "<#") && strings.HasSuffix(args[2], ">") {
				chstr = args[2]
				chstr = chstr[2 : len(chstr)-1]
			} else {
				chstr = args[2]
			}
			channel, err = s.State.Channel(chstr)
			if err != nil {
				s.ChannelMessageSend(ch.ID, "no")
				return
			}
		}
		switch strings.ToLower(args[1]) {
		case "join":
			b.db.Exec("UPDATE discordguilds SET joinlog=$1 WHERE guildid=$2", channel.ID, g.ID)
			s.ChannelMessageSend(ch.ID, fmt.Sprintf("Set join logs to %v.", channel.Mention()))
		case "leave":
			b.db.Exec("UPDATE discordguilds SET leavelog=$1 WHERE guildid=$2", channel.ID, g.ID)
			s.ChannelMessageSend(ch.ID, fmt.Sprintf("Set leave logs to %v.", channel.Mention()))
		case "msgdelete":
			b.db.Exec("UPDATE discordguilds SET msgdeletelog=$1 WHERE guildid=$2", channel.ID, g.ID)
			s.ChannelMessageSend(ch.ID, fmt.Sprintf("Set message delete logs to %v.", channel.Mention()))
		case "msgedit":
			b.db.Exec("UPDATE discordguilds SET msgeditlog=$1 WHERE guildid=$2", channel.ID, g.ID)
			s.ChannelMessageSend(ch.ID, fmt.Sprintf("Set message edit logs to %v.", channel.Mention()))
		case "ban":
			b.db.Exec("UPDATE discordguilds SET banlog=$1 WHERE guildid=$2", channel.ID, g.ID)
			s.ChannelMessageSend(ch.ID, fmt.Sprintf("Set ban logs to %v.", channel.Mention()))
		case "unban":
			b.db.Exec("UPDATE discordguilds SET unbanlog=$1 WHERE guildid=$2", channel.ID, g.ID)
			s.ChannelMessageSend(ch.ID, fmt.Sprintf("Set unban logs to %v.", channel.Mention()))

		default:
			s.ChannelMessageSend(ch.ID, "Please choose a valid log type.")
		}
	} else if args[0] == "fl.help" {

		text := "To set a log channel, do `fl.set <logtype> <channel>`, where channel is optional.\n"
		text += "Log types:\n"
		text += "Join - When a user joins the server\n"
		text += "Leave - When a user leaves the server\n"
		text += "Msgdelete - When a message is deleted\n"
		text += "Msgedit - When a message is edited\n"
		text += "Ban - When a user got banned\n"
		text += "Unban - When a user got unbanned\n"
		text += "\n"
		text += "Example - fl.set join\n"
		text += "Example - fl.set join #join-logs\n"
		text += "Example - fl.set join 1234123412341234\n"

		s.ChannelMessageSend(ch.ID, text)
	}

}
