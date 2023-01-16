package bot

import (
	"bytes"
	"fmt"
	"github.com/dgraph-io/badger"
	"github.com/intrntsrfr/functional-logger/kvstore"
	"math"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

func (b *Bot) readyHandler(s *discordgo.Session, r *discordgo.Ready) {
	statusTimer := time.NewTicker(time.Second * 15)

	go func() {
		// run every 15 seconds
		i := 0
		for range statusTimer.C {
			switch i {
			case 0:
				_ = s.UpdateGameStatus(0, "fl.help")
			case 1:
				_ = s.UpdateListeningStatus("lots of events")
			}

			i = (i + 1) % 2
		}
	}()

	fmt.Println("Logged in as", r.User.String())
}

func (b *Bot) disconnectHandler(s *discordgo.Session, d *discordgo.Disconnect) {
	//atomic.StoreInt64(&b.loggerDB.TotalMembers, 0)
	fmt.Println("DISCONNECTED AT ", time.Now().Format(time.RFC3339))
}

func (b *Bot) guildBanAddHandler(s *discordgo.Session, m *discordgo.GuildBanAdd) {
	gc, err := b.db.GetGuild(m.GuildID)
	if err != nil {
		return
	}
	if gc.BanLog == "" {
		return
	}

	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		b.log.Info("failed to fetch guild", zap.Error(err))
		return
	}

	embed := discordgo.MessageEmbed{
		Color: int(Red),
		Title: "User banned",
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: m.User.AvatarURL("256"),
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "User",
				Value: fmt.Sprintf("%v\n%v", m.User.Mention(), m.User.String()),
			},
			{
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

	if _, err = b.store.GetMember(fmt.Sprintf("%v:%v", m.GuildID, m.User.ID)); err != nil {
		if err != badger.ErrKeyNotFound {
			return
		}
		// if user is not found, aka they were never in the server
		embed.Title += " - Hackban"

		_, err = s.ChannelMessageSendEmbed(gc.BanLog, &embed)
		if err != nil {
			b.log.Info("failed to send log message", zap.Error(err))
		}
		return
	}

	messageLog, err := b.store.GetMessageLog(m)
	if err != nil {
		return
	}

	text := strings.Builder{}
	sort.Sort(ByID(messageLog))

	for _, cmsg := range messageLog {
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

	if len(messageLog) > 0 {
		if b.owo != nil {
			link, err := b.owo.Upload(text.String())
			if err != nil {
				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
					Name:  "24h user log",
					Value: "Error getting link",
				})
				b.log.Info("BAN LOG ERROR", zap.Error(err))
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
			b.log.Info("error", zap.Error(err))
			fmt.Println("BAN LOG ERROR", err)
		}
	}
}

func (b *Bot) guildBanRemoveHandler(s *discordgo.Session, m *discordgo.GuildBanRemove) {

	dg := Guild{}
	b.db.Get(&dg, "SELECT unban_log FROM guilds WHERE id=$1;", m.GuildID)
	if dg.UnbanLog == "" {
		return
	}

	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		b.log.Info("error", zap.Error(err))
		return
	}

	embed := discordgo.MessageEmbed{
		Color: int(Green),
		Title: "User unbanned",
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: m.User.AvatarURL("256"),
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "User",
				Value: fmt.Sprintf("%v\n%v", m.User.Mention(), m.User.String()),
			},
			{
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
		b.log.Info("error", zap.Error(err))
		fmt.Println("UNBAN LOG ERROR", err)
	}
}

func (b *Bot) guildCreateHandler(s *discordgo.Session, g *discordgo.GuildCreate) {
	if _, err := b.db.GetGuild(g.ID); err != nil {
		err = b.db.CreateGuild(g.ID)
		if err != nil {
			b.log.Error("failed to create new guild", zap.Error(err))
		}
	}

	if len(g.Members) != g.MemberCount {
		_ = s.RequestGuildMembers(g.ID, "", 0, "", false)
		return
	}

	for _, mem := range g.Members {
		err := b.store.SetMember(mem, 1)
		if err != nil {
			b.log.Error("failed to set member", zap.Error(err))
			continue
		}
	}

	b.log.Info("guild created", zap.String("name", g.Name))
}

func (b *Bot) guildMemberAddHandler(s *discordgo.Session, m *discordgo.GuildMemberAdd) {

	dg := Guild{}
	b.db.Get(&dg, "SELECT join_log FROM guilds WHERE id=$1;", m.GuildID)
	if dg.JoinLog == "" {
		return
	}

	err := b.store.SetMember(m.Member, 1)
	if err != nil {
		fmt.Println(err)
		b.log.Info("error", zap.Error(err))
		return
	}
	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		b.log.Info("error", zap.Error(err))
		return
	}

	id, err := strconv.ParseInt(m.User.ID, 0, 63)
	if err != nil {
		b.log.Info("error", zap.Error(err))
		return
	}

	id = ((id >> 22) + 1420070400000) / 1000

	dur := time.Since(time.Unix(int64(id), 0))

	ts := time.Unix(id, 0)

	embed := discordgo.MessageEmbed{
		Color: int(Blue),
		Title: "User joined",
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: m.User.AvatarURL("256"),
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "User",
				Value: fmt.Sprintf("%v\n%v (%v)", m.User.Mention(), m.User.String(), m.User.ID),
			},
			{
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

	_, err = s.ChannelMessageSendEmbed(dg.JoinLog, &embed)
	if err != nil {
		b.log.Info("error", zap.Error(err))
		fmt.Println("JOIN LOG ERROR", err)
	}
}

func (b *Bot) guildMemberRemoveHandler(s *discordgo.Session, m *discordgo.GuildMemberRemove) {
	gc, err := b.db.GetGuild(m.GuildID)
	if gc.LeaveLog == "" {
		return
	}

	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		return
	}

	mem, err := b.store.GetMember(fmt.Sprintf("%v:%v", m.GuildID, m.User.ID))
	if err != nil {
		return
	}

	var roles []string
	for _, r := range mem.Roles {
		roles = append(roles, fmt.Sprintf("<@&%v>", r))
	}

	embed := discordgo.MessageEmbed{
		Color: Orange,
		Title: "User left or kicked",
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: m.User.AvatarURL("256"),
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "User",
				Value:  fmt.Sprintf("%v\n%v", m.User.Mention(), m.User.String()),
				Inline: true,
			},
			{
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
		b.log.Info("error", zap.Error(err))
		fmt.Println("LEAVE LOG ERROR", err)
	}

	err = b.store.DeleteMember(fmt.Sprintf("%v:%v", m.GuildID, m.User.ID))
	if err != nil {
		b.log.Info("error", zap.Error(err))
		fmt.Println(err)
		return
	}
}

func (b *Bot) guildMembersChunkHandler(s *discordgo.Session, g *discordgo.GuildMembersChunk) {

	go func() {
		for _, mem := range g.Members {

			err := b.store.SetMember(mem)
			if err != nil {
				b.log.Error("error", zap.Error(err))
				continue
			}
		}

		/*
			sg, err := s.State.Guild(g.GuildID)
			if err != nil {
				b.logger.Error("error", zap.Error(err))
				return
			}
			b.logger.Info(fmt.Sprintf("UPDATED %v MEMBERS", sg.Name))
			fmt.Println(fmt.Sprintf("UPDATED %v MEMBERS", sg.Name))
		*/
	}()
}

func (b *Bot) guildMemberUpdateHandler(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {

	err := b.store.SetMember(m.Member)
	if err != nil {
		fmt.Println(err)
		b.log.Info("error", zap.Error(err))
		return
	}
}

func (b *Bot) messageCreateHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		return
	}

	ch, err := s.State.Channel(m.ChannelID)
	if err != nil {
		return
	}

	// max size 10mb
	b.store.SetMessage(kvstore.NewDiscordMessage(m.Message, 1024*1024*10))

	if strings.HasPrefix(m.Content, "fl.info") {
		_, _ = s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
			Title:       "Info",
			Description: fmt.Sprintf("Golang version: %v", runtime.Version()),
			Color:       Blue,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Golang version",
					Value: runtime.Version(),
				},
				{
					Name:  "Running since",
					Value: fmt.Sprintf("<t:%v:R>", b.startTime.Unix()),
				},
			},
		})
		return
	}

	args := strings.Split(m.Content, " ")

	uperms, err := s.State.UserChannelPermissions(m.Author.ID, ch.ID)
	if err != nil {
		return
	}

	if args[0] == "fl.set" {
		if len(args) < 2 {
			return
		}

		if uperms&discordgo.PermissionAdministrator == 0 {
			s.ChannelMessageSend(ch.ID, "You need admin perms to do this")
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
			isServerChannel := false

			for _, gChannel := range g.Channels {
				if gChannel.ID == channel.ID {
					isServerChannel = true
				}
			}

			if !isServerChannel {
				s.ChannelMessageSend(ch.ID, "nice try")
				return
			}

		}
		switch strings.ToLower(args[1]) {
		case "join":
			b.db.Exec("UPDATE Guilds SET joinlog=$1 WHERE guildid=$2", channel.ID, g.ID)
			s.ChannelMessageSend(ch.ID, fmt.Sprintf("Set join logs to %v.", channel.Mention()))
		case "leave":
			b.db.Exec("UPDATE Guilds SET leavelog=$1 WHERE guildid=$2", channel.ID, g.ID)
			s.ChannelMessageSend(ch.ID, fmt.Sprintf("Set leave logs to %v.", channel.Mention()))
		case "msgdelete":
			b.db.Exec("UPDATE Guilds SET msgdeletelog=$1 WHERE guildid=$2", channel.ID, g.ID)
			s.ChannelMessageSend(ch.ID, fmt.Sprintf("Set message delete logs to %v.", channel.Mention()))
		case "msgedit":
			b.db.Exec("UPDATE Guilds SET msgeditlog=$1 WHERE guildid=$2", channel.ID, g.ID)
			s.ChannelMessageSend(ch.ID, fmt.Sprintf("Set message edit logs to %v.", channel.Mention()))
		case "ban":
			b.db.Exec("UPDATE Guilds SET banlog=$1 WHERE guildid=$2", channel.ID, g.ID)
			s.ChannelMessageSend(ch.ID, fmt.Sprintf("Set ban logs to %v.", channel.Mention()))
		case "unban":
			b.db.Exec("UPDATE Guilds SET unbanlog=$1 WHERE guildid=$2", channel.ID, g.ID)
			s.ChannelMessageSend(ch.ID, fmt.Sprintf("Set unban logs to %v.", channel.Mention()))
		}
	} else if args[0] == "fl.help" {
		text := strings.Builder{}
		text.WriteString("To set a log channel, do `fl.set <logtype> <channel>`, where channel is optional.\n")
		text.WriteString("Log types:\n")
		text.WriteString("`join` - When a user joins the server\n")
		text.WriteString("`leave` - When a user leaves the server\n")
		text.WriteString("`msgdelete` - When a message is deleted\n")
		text.WriteString("`msgedit` - When a message is edited\n")
		text.WriteString("`ban` - When a user got banned\n")
		text.WriteString("`unban` - When a user got unbanned\n")
		text.WriteString("\n")
		text.WriteString("Example - fl.set join\n")
		text.WriteString("Example - fl.set join #join-logs\n")
		text.WriteString("Example - fl.set join 1234123412341234\n")
		s.ChannelMessageSend(ch.ID, text.String())
	}

}

func (b *Bot) messageDeleteHandler(s *discordgo.Session, m *discordgo.MessageDelete) {

	dg := Guild{}
	b.db.Get(&dg, "SELECT msg_delete_log FROM guilds WHERE id=$1;", m.GuildID)
	if dg.MsgDeleteLog == "" {
		return
	}

	msg, err := b.store.GetMessage(fmt.Sprintf("%v:%v:%v", m.GuildID, m.ChannelID, m.ID))
	if err != nil {
		//fmt.Println(err)
		return
	}
	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		b.log.Info("error", zap.Error(err))
		return
	}

	embed := discordgo.MessageEmbed{
		Color: int(White),
		Title: "Message deleted",
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "User",
				Value:  fmt.Sprintf("%v\n%v\n%v", msg.Message.Author.Mention(), msg.Message.Author.String(), msg.Message.Author.ID),
				Inline: true,
			},
			{
				Name:   "Message ID",
				Value:  m.ID,
				Inline: true,
			},
			{
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

	if msg.Message.Content != "" {
		content := ""
		if len(msg.Message.Content) > 1024 {
			link, err := b.owo.Upload(msg.Message.Content)
			if err != nil {
				content = "Content unavailable"
			} else {
				content = "Message too big for embed, have a link instead: " + link
			}
		} else {
			content = msg.Message.Content
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Content",
			Value: content,
		})
	} else {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Content",
			Value: "No content",
		})
	}

	if len(msg.Message.Attachments) > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Total attachments",
			Value: fmt.Sprint(len(msg.Message.Attachments)),
		})
	}

	_, err = s.ChannelMessageSendEmbed(dg.MsgDeleteLog, &embed)
	if err != nil {
		b.log.Info("error", zap.Error(err))
		fmt.Println("DELETE LOG ERROR", err)
	}
	if len(msg.Message.Attachments) > 0 {
		send, err := s.ChannelMessageSend(dg.MsgDeleteLog, "Trying to get attachments..")
		if err != nil {
			b.log.Info("error", zap.Error(err))
			fmt.Println("DELETE LOG SEND ERROR", err)
			return
		}

		data := &discordgo.MessageSend{
			Content: fmt.Sprintf("File(s) attached to message ID: %v", m.ID),
		}

		for k, img := range msg.Attachments {
			f := &discordgo.File{
				Name:   msg.Message.Attachments[k].Filename,
				Reader: bytes.NewReader(img),
			}
			data.Files = append(data.Files, f)
		}

		_, err = s.ChannelMessageSendComplex(dg.MsgDeleteLog, data)
		if err != nil {
			b.log.Info("error", zap.Error(err))
			s.ChannelMessageEdit(send.ChannelID, send.ID, "Error getting attachments")
			fmt.Println("DELETE LOG ERROR", err)
		} else {
			s.ChannelMessageDelete(send.ChannelID, send.ID)
		}
	}
}

func (b *Bot) messageDeleteBulkHandler(s *discordgo.Session, m *discordgo.MessageDeleteBulk) {

	dg := Guild{}
	b.db.Get(&dg, "SELECT msg_delete_log FROM guilds WHERE id=$1;", m.GuildID)
	if dg.MsgDeleteLog == "" {
		return
	}

	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		b.log.Info("error", zap.Error(err))
		return
	}
	ts := time.Now()

	embed := discordgo.MessageEmbed{
		Color: int(White),
		Title: fmt.Sprintf("Bulk message delete - (%v) messages deleted", len(m.Messages)),
		Fields: []*discordgo.MessageEmbedField{
			{
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

	deletedmsgs := []*DiscordMessage{}
	for _, msgid := range m.Messages {
		delmsg, err := b.store.GetMessage(fmt.Sprintf("%v:%v:%v", m.GuildID, m.ChannelID, msgid))
		if err != nil {
			b.log.Info("error", zap.Error(err))
			continue
		}
		deletedmsgs = append(deletedmsgs, delmsg)
	}

	sort.Sort(ByID(deletedmsgs))

	text := strings.Builder{}
	text.WriteString(fmt.Sprintf("%v - %v\n\n\n", m.ChannelID, ts.Format(time.RFC1123)))

	for _, msg := range deletedmsgs {
		if len(msg.Attachments) > 0 {
			text.WriteString(fmt.Sprintf("\nUser: %v (%v)\nContent: %v\nMessage had attachment\n", msg.Message.Author.String(), msg.Message.Author.ID, msg.Message.Content))
		} else {
			text.WriteString(fmt.Sprintf("\nUser: %v (%v)\nContent: %v\n", msg.Message.Author.String(), msg.Message.Author.ID, msg.Message.Content))
		}
	}

	if b.owo != nil {
		res, err := b.owo.Upload(text.String())

		if err != nil {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  "Logged messages:",
				Value: "Error getting link",
			})
			b.log.Info("BULK DELETE LOG ERROR", zap.Error(err))
		} else {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  "Logged messages:",
				Value: "Link: " + res,
			})
		}
		_, err = s.ChannelMessageSendEmbed(dg.MsgDeleteLog, &embed)
		if err != nil {
			fmt.Println("BULK DELETE LOG ERROR", err)
		}
	} else {
		jeff := bytes.Buffer{}
		jeff.WriteString(text.String())

		msg, err := s.ChannelMessageSendEmbed(dg.MsgDeleteLog, &embed)
		if err != nil {
			fmt.Println("BULK DELETE LOG ERROR", err)
		}

		s.ChannelFileSendWithMessage(dg.MsgDeleteLog, fmt.Sprintf("Log file for delete log message ID %v:", msg.ID), "deletelog_"+m.ChannelID+".txt", &jeff)
	}
}

func (b *Bot) messageUpdateHandler(s *discordgo.Session, m *discordgo.MessageUpdate) {

	dg := Guild{}
	err := b.db.Get(&dg, "SELECT msg_edit_log FROM guilds WHERE id=$1;", m.GuildID)
	if err != nil {
		fmt.Println(err)
		return
	}

	if dg.MsgEditLog == "" {
		return
	}

	// This means it was an image update and not an actual edit
	if m.Message.Content == "" {
		return
	}

	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		b.log.Info("error", zap.Error(err))
		return
	}

	oldm, err := b.store.GetMessage(fmt.Sprintf("%v:%v:%v", m.GuildID, m.ChannelID, m.ID))
	if err != nil {
		return
	}

	oldmsg := oldm.Message

	if oldmsg.Content == m.Content {
		return
	}

	oldc := ""
	newc := ""

	if len(m.Content) > 1024 {
		link, err := b.owo.Upload(m.Content)
		if err != nil {
			newc = "Content unavailable"
		} else {
			newc = "Message too big for embed, have a link instead: " + link
		}
	} else {
		newc = m.Content
	}

	if len(oldmsg.Content) > 1024 {
		link, err := b.owo.Upload(oldmsg.Content)
		if err != nil {
			oldc = "Content unavailable"
		} else {
			oldc = "Message too big for embed, have a link instead: " + link
		}
	} else {
		oldc = oldmsg.Content
	}

	embed := discordgo.MessageEmbed{
		Color: int(Blue),
		Title: "Message edited",
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "User",
				Value:  fmt.Sprintf("%v\n%v\n%v", oldmsg.Author.Mention(), oldmsg.Author.String(), oldmsg.Author.ID),
				Inline: true,
			},
			{
				Name:   "Message ID",
				Value:  m.ID,
				Inline: true,
			},
			{
				Name:  "Channel",
				Value: fmt.Sprintf("<#%v> (%v)", m.ChannelID, m.ChannelID),
			},
			{
				Name:  "Old content",
				Value: oldc,
			},
			{
				Name:  "New content",
				Value: newc,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			IconURL: discordgo.EndpointGuildIcon(g.ID, g.Icon),
			Text:    g.Name,
		},
	}

	_, err = s.ChannelMessageSendEmbed(dg.MsgEditLog, &embed)
	if err != nil {
		b.log.Info("error", zap.Error(err))
		fmt.Println("EDIT LOG ERROR", err)
	}

	oldm.Message.Content = m.Content

	err = b.store.SetMessage(oldm.Message)
	if err != nil {
		b.log.Info("error", zap.Error(err))
		fmt.Println("ERROR")
		return
	}
}
