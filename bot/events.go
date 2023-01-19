package bot

import (
	"bytes"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/dgraph-io/badger"
	"github.com/intrntsrfr/functional-logger/kvstore"
	"go.uber.org/zap"
	"runtime"
	"sort"
	"strings"
	"time"
)

func readyHandler(c *Context, r *discordgo.Ready) {
	statusTimer := time.NewTicker(time.Second * 15)

	go func() {
		// run every 15 seconds
		i := 0
		for range statusTimer.C {
			switch i {
			case 0:
				_ = c.s.UpdateGameStatus(0, "fl.help")
			case 1:
				_ = c.s.UpdateListeningStatus("lots of events")
			}

			i = (i + 1) % 2
		}
	}()

	fmt.Println("Logged in as", r.User.String())
}

func disconnectHandler(c *Context, _ *discordgo.Disconnect) {
	c.b.log.Info("disconnected")
}

func guildBanAddHandler(c *Context, m *discordgo.GuildBanAdd) {
	g, err := c.s.State.Guild(m.GuildID)
	if err != nil {
		c.b.log.Info("failed to fetch guild", zap.Error(err))
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

	if _, err = c.b.store.GetMember(fmt.Sprintf("%v:%v", m.GuildID, m.User.ID)); err != nil {
		if err != badger.ErrKeyNotFound {
			return
		}
		// if user is not found, aka they were never in the server
		embed.Title += " - Hackban"

		_, err = c.s.ChannelMessageSendEmbed(c.gc.BanLog, &embed)
		if err != nil {
			c.b.log.Info("failed to send log message", zap.Error(err))
		}
		return
	}

	messageLog, err := c.b.store.GetMessageLog(m)
	if err != nil {
		return
	}

	text := strings.Builder{}
	sort.Sort(kvstore.ByID(messageLog))

	for _, cmsg := range messageLog {
		if cmsg.Message.Author.ID == m.User.ID {

			ch, err := c.s.State.Channel(cmsg.Message.ChannelID)
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
		if c.b.owo != nil {
			link, err := c.b.owo.Upload(text.String())
			if err != nil {
				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
					Name:  "24h user log",
					Value: "Error getting link",
				})
				c.b.log.Info("BAN LOG ERROR", zap.Error(err))
			} else {
				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
					Name:  "24h user log",
					Value: "Link: " + link,
				})
			}

			_, err = c.s.ChannelMessageSendEmbed(c.gc.BanLog, &embed)
			if err != nil {
				fmt.Println("BAN LOG ERROR", err)
			}
		} else {
			buf := bytes.Buffer{}
			buf.WriteString(text.String())

			msg, err := c.s.ChannelMessageSendEmbed(c.gc.BanLog, &embed)
			if err != nil {
				fmt.Println("BAN LOG ERROR", err)
			}

			_, _ = c.s.ChannelMessageSendComplex(c.gc.BanLog, &discordgo.MessageSend{
				Content: fmt.Sprintf("Log file for delete log message ID %v:", msg.ID),
				File: &discordgo.File{
					Name:   fmt.Sprintf("banlog_%v.txt", m.User.ID),
					Reader: &buf,
				},
			})
		}
	} else {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "24h user log",
			Value: "No history.",
		})

		_, _ = c.s.ChannelMessageSendEmbed(c.gc.BanLog, &embed)
	}
}

func guildBanRemoveHandler(c *Context, m *discordgo.GuildBanRemove) {
	g, err := c.s.State.Guild(m.GuildID)
	if err != nil {
		return
	}

	embed := discordgo.MessageEmbed{
		Color: Green,
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
	_, _ = c.s.ChannelMessageSendEmbed(c.gc.UnbanLog, &embed)
}

func guildCreateHandler(c *Context, g *discordgo.GuildCreate) {
	if _, err := c.b.db.GetGuild(g.ID); err != nil {
		err = c.b.db.CreateGuild(g.ID)
		if err != nil {
			c.b.log.Error("failed to create new guild", zap.Error(err))
		}
	}

	if len(g.Members) != g.MemberCount {
		_ = c.s.RequestGuildMembers(g.ID, "", 0, "", false)
		return
	}

	for _, mem := range g.Members {
		err := c.b.store.SetMember(mem)
		if err != nil {
			c.b.log.Error("failed to set member", zap.Error(err))
			continue
		}
	}

	c.b.log.Info("guild created", zap.String("name", g.Name))
}

func guildMemberAddHandler(c *Context, m *discordgo.GuildMemberAdd) {
	err := c.b.store.SetMember(m.Member)
	if err != nil {
		c.b.log.Error("failed to set member", zap.Error(err))
	}

	g, err := c.s.State.Guild(m.GuildID)
	if err != nil {
		return
	}

	ts, err := ParseSnowflake(m.User.ID)
	embed := discordgo.MessageEmbed{
		Color: Blue,
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
				Value: fmt.Sprintf("<t:%v:R>", ts),
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			IconURL: discordgo.EndpointGuildIcon(g.ID, g.Icon),
			Text:    g.Name,
		},
	}

	_, _ = c.s.ChannelMessageSendEmbed(c.gc.JoinLog, &embed)
}

func guildMemberRemoveHandler(c *Context, m *discordgo.GuildMemberRemove) {
	g, err := c.s.State.Guild(m.GuildID)
	if err != nil {
		return
	}

	mem, err := c.b.store.GetMember(fmt.Sprintf("%v:%v", m.GuildID, m.User.ID))
	if err != nil {
		return
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

	var roles []string
	for _, r := range mem.Roles {
		roles = append(roles, fmt.Sprintf("<@&%v>", r))
	}

	if len(roles) <= 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Roles",
			Value: "None",
		})
	} else {
		var shown []string
		for _, r := range roles {
			if len(strings.Join(append(shown, r), ", ")) > 760 {
				break
			}
			shown = append(shown, r)
		}

		embedStr := strings.Join(shown, ", ")
		if len(shown) != len(roles) {
			embedStr += fmt.Sprintf(" and %v more", len(roles)-len(shown))
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Roles",
			Value: embedStr,
		})
	}

	_, _ = c.s.ChannelMessageSendEmbed(c.gc.LeaveLog, &embed)

	err = c.b.store.DeleteMember(fmt.Sprintf("%v:%v", m.GuildID, m.User.ID))
	if err != nil {
		c.b.log.Error("failed to delete member", zap.Error(err))
	}
}

func guildMembersChunkHandler(c *Context, g *discordgo.GuildMembersChunk) {
	for _, mem := range g.Members {
		err := c.b.store.SetMember(mem)
		if err != nil {
			c.b.log.Error("failed to set member", zap.Error(err))
			continue
		}
	}
}

func guildMemberUpdateHandler(c *Context, m *discordgo.GuildMemberUpdate) {
	err := c.b.store.SetMember(m.Member)
	if err != nil {
		c.b.log.Error("failed to update member", zap.Error(err))
		return
	}
}

func messageCreateHandler(c *Context, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	g, err := c.s.State.Guild(m.GuildID)
	if err != nil {
		return
	}

	ch, err := c.s.State.Channel(m.ChannelID)
	if err != nil {
		return
	}

	// max size 10mb
	_ = c.b.store.SetMessage(kvstore.NewDiscordMessage(m.Message, 1024*1024*10))

	if strings.HasPrefix(m.Content, "fl.info") {
		_, _ = c.s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
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
					Value: fmt.Sprintf("<t:%v:R>", c.b.startTime.Unix()),
				},
			},
		})
		return
	}

	args := strings.Split(m.Content, " ")
	uperms, err := c.s.State.UserChannelPermissions(m.Author.ID, ch.ID)
	if err != nil {
		return
	}

	if args[0] == "fl.set" {
		if len(args) < 2 {
			return
		}

		gc, err := c.b.db.GetGuild(g.ID)
		if err != nil {
			c.b.log.Error("failed to get guild config", zap.Error(err))
			return
		}

		if uperms&discordgo.PermissionAdministrator == 0 {
			_, _ = c.s.ChannelMessageSend(ch.ID, "This is admin only, sorry!")
			return
		}

		setChannel := ch
		if len(args) >= 3 {
			chStr := TrimChannelString(args[2])
			setChannel, err = c.s.State.Channel(chStr)
			if err != nil || setChannel.GuildID != g.ID {
				return
			}
		}
		switch strings.ToLower(args[1]) {
		case "join":
			gc.JoinLog = setChannel.ID
		case "leave":
			gc.LeaveLog = setChannel.ID
		case "msgdelete":
			gc.MsgDeleteLog = setChannel.ID
		case "msgedit":
			gc.MsgEditLog = setChannel.ID
		case "ban":
			gc.BanLog = setChannel.ID
		case "unban":
			gc.UnbanLog = setChannel.ID
		}

		err = c.b.db.UpdateGuild(g.ID, gc)
		if err != nil {
			_, _ = c.s.ChannelMessageSend(ch.ID, "Could not update config ")
			c.b.log.Error("failed to update guild config", zap.Error(err))
			return
		}

		_, _ = c.s.ChannelMessageSend(ch.ID, "Updated config")
	} else if args[0] == "fl.help" {
		text := strings.Builder{}
		text.WriteString("To set a log channel, do `fl.set [logtype] <channel>`, where channel is optional.\n")
		text.WriteString("Logtypes:\n")
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
		_, _ = c.s.ChannelMessageSend(ch.ID, text.String())
	}
}

/*
func messageDeleteHandler(c*Context, m *discordgo.MessageDelete) {

	gc, err := c.b.db.GetGuild(m.GuildID)
	if err != nil || gc.MsgDeleteLog == "" {
		return
	}

	msg, err := c.b.store.GetMessage(fmt.Sprintf("%v:%v:%v", m.GuildID, m.ChannelID, m.ID))
	if err != nil {
		return
	}
	g, err := c.s.State.gc(m.GuildID)
	if err != nil {
	 	c.b.log.Info("error", zap.Error(err))
		return
	}

	embed := discordgo.MessageEmbed{
		Color: White,
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
			link, err := c.b.owo.Upload(msg.Message.Content)
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

	_, err = c.s.ChannelMessageSendEmbed(dg.MsgDeleteLog, &embed)
	if err != nil {
	 	c.b.log.Info("error", zap.Error(err))
		fmt.Println("DELETE LOG ERROR", err)
	}
	if len(msg.Message.Attachments) > 0 {
		send, err := c.s.ChannelMessageSend(dg.MsgDeleteLog, "Trying to get attachments..")
		if err != nil {
		 	c.b.log.Info("error", zap.Error(err))
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

		_, err = c.s.ChannelMessageSendComplex(dg.MsgDeleteLog, data)
		if err != nil {
		 	c.b.log.Info("error", zap.Error(err))
			s.ChannelMessageEdit(send.ChannelID, send.ID, "Error getting attachments")
			fmt.Println("DELETE LOG ERROR", err)
		} else {
			s.ChannelMessageDelete(send.ChannelID, send.ID)
		}
	}
}

func (b *Bot) messageDeleteBulkHandler(s *discordgo.Session, m *discordgo.MessageDeleteBulk) {

	gc, err := c.b.db.GetGuild(m.GuildID)
	if err != nil || gc.MsgDeleteLog == "" {
		return
	}

	g, err := c.s.State.gc(m.GuildID)
	if err != nil {
	 	c.b.log.Info("error", zap.Error(err))
		return
	}
	ts := time.Now()

	embed := discordgo.MessageEmbed{
		Color: White,
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
		delmsg, err := c.b.store.GetMessage(fmt.Sprintf("%v:%v:%v", m.GuildID, m.ChannelID, msgid))
		if err != nil {
		 	c.b.log.Info("error", zap.Error(err))
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

	if c.b.owo != nil {
		res, err := c.b.owo.Upload(text.String())

		if err != nil {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  "Logged messages:",
				Value: "Error getting link",
			})
		 	c.b.log.Info("BULK DELETE LOG ERROR", zap.Error(err))
		} else {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  "Logged messages:",
				Value: "Link: " + res,
			})
		}
		_, err = c.s.ChannelMessageSendEmbed(dg.MsgDeleteLog, &embed)
		if err != nil {
			fmt.Println("BULK DELETE LOG ERROR", err)
		}
	} else {
		jeff := bytes.Buffer{}
		jeff.WriteString(text.String())

		msg, err := c.s.ChannelMessageSendEmbed(dg.MsgDeleteLog, &embed)
		if err != nil {
			fmt.Println("BULK DELETE LOG ERROR", err)
		}

		s.ChannelFileSendWithMessage(dg.MsgDeleteLog, fmt.Sprintf("Log file for delete log message ID %v:", msg.ID), "deletelog_"+m.ChannelID+".txt", &jeff)
	}
}

func (b *Bot) messageUpdateHandler(s *discordgo.Session, m *discordgo.MessageUpdate) {

	gc, err := c.b.db.GetGuild(m.GuildID)
	if err != nil || gc.MsgEditLog == "" {
		return
	}

	// This means it was an image update and not an actual edit
	if m.Message.Content == "" {
		return
	}

	g, err := c.s.State.gc(m.GuildID)
	if err != nil {
	 	c.b.log.Info("error", zap.Error(err))
		return
	}

	oldm, err := c.b.store.GetMessage(fmt.Sprintf("%v:%v:%v", m.GuildID, m.ChannelID, m.ID))
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
		link, err := c.b.owo.Upload(m.Content)
		if err != nil {
			newc = "Content unavailable"
		} else {
			newc = "Message too big for embed, have a link instead: " + link
		}
	} else {
		newc = m.Content
	}

	if len(oldmsg.Content) > 1024 {
		link, err := c.b.owo.Upload(oldmsg.Content)
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

	_, err = c.s.ChannelMessageSendEmbed(dg.MsgEditLog, &embed)
	if err != nil {
	 	c.b.log.Info("error", zap.Error(err))
		fmt.Println("EDIT LOG ERROR", err)
	}

	oldm.Message.Content = m.Content

	err = c.b.store.SetMessage(oldm.Message)
	if err != nil {
	 	c.b.log.Info("error", zap.Error(err))
		fmt.Println("ERROR")
		return
	}
}

*/
