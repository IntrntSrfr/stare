package bot

/*
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
	b.log.Info("disconnected")
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
	sort.Sort(kvstore.ByID(messageLog))

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

			_, err = s.ChannelMessageSendEmbed(gc.BanLog, &embed)
			if err != nil {
				fmt.Println("BAN LOG ERROR", err)
			}
		} else {
			jeff := bytes.Buffer{}
			jeff.WriteString(text.String())

			msg, err := s.ChannelMessageSendEmbed(gc.BanLog, &embed)
			if err != nil {
				fmt.Println("BAN LOG ERROR", err)
			}

			s.ChannelFileSendWithMessage(gc.BanLog, fmt.Sprintf("Log file for delete log message ID %v:", msg.ID), "banlog_"+m.User.ID+".txt", &jeff)
		}
	} else {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "24h user log",
			Value: "No history.",
		})

		_, _ = s.ChannelMessageSendEmbed(gc.BanLog, &embed)
	}
}

func (b *Bot) guildBanRemoveHandler(s *discordgo.Session, m *discordgo.GuildBanRemove) {
	gc, err := b.db.GetGuild(m.GuildID)
	if err != nil || gc.UnbanLog == "" {
		return
	}

	g, err := s.State.Guild(m.GuildID)
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
	_, _ = s.ChannelMessageSendEmbed(gc.UnbanLog, &embed)
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
		err := b.store.SetMember(mem)
		if err != nil {
			b.log.Error("failed to set member", zap.Error(err))
			continue
		}
	}

	b.log.Info("guild created", zap.String("name", g.Name))
}

func (b *Bot) guildMemberAddHandler(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	gc, err := b.db.GetGuild(m.GuildID)
	if err != nil || gc.JoinLog == "" {
		return
	}

	err = b.store.SetMember(m.Member)
	if err != nil {
		b.log.Error("failed to set member", zap.Error(err))
	}

	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		return
	}

	id, err := strconv.ParseInt(m.User.ID, 0, 63)
	if err != nil {
		return
	}
	ts := time.Unix(((id>>22)+1420070400000)/1000, 0)

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
				Value: fmt.Sprintf("<t:%v:R>", ts),
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			IconURL: discordgo.EndpointGuildIcon(g.ID, g.Icon),
			Text:    g.Name,
		},
	}

	_, _ = s.ChannelMessageSendEmbed(gc.JoinLog, &embed)
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
		shown := []string{}
		for _, r := range roles {
			if len(strings.Join(append(shown, r), ", ")) > 760 {
				break
			}
			shown = append(shown, r)
		}

		embedStr := strings.Join(shown, ", ")
		if len(shown) != len(roles) {
			embedStr += fmt.Sprint(" and %v more", len(roles)-len(shown))
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Roles",
			Value: embedStr,
		})
	}

	_, _ = s.ChannelMessageSendEmbed(gc.LeaveLog, &embed)

	err = b.store.DeleteMember(fmt.Sprintf("%v:%v", m.GuildID, m.User.ID))
	if err != nil {
		b.log.Error("failed to delete member", zap.Error(err))
	}
}

func (b *Bot) guildMembersChunkHandler(s *discordgo.Session, g *discordgo.GuildMembersChunk) {
	for _, mem := range g.Members {
		err := b.store.SetMember(mem)
		if err != nil {
			b.log.Error("failed to set member", zap.Error(err))
			continue
		}
	}
}

func (b *Bot) guildMemberUpdateHandler(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {
	err := b.store.SetMember(m.Member)
	if err != nil {
		b.log.Error("failed to update member", zap.Error(err))
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
	_ = b.store.SetMessage(kvstore.NewDiscordMessage(m.Message, 1024*1024*10))

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

		gc, err := b.db.GetGuild(g.ID)
		if err != nil {
			b.log.Error("failed to get guild config", zap.Error(err))
			return
		}

		if uperms&discordgo.PermissionAdministrator == 0 {
			s.ChannelMessageSend(ch.ID, "This is admin only, sorry!")
			return
		}

		setChannel := ch
		if len(args) >= 3 {
			chStr := TrimChannelString(args[2])
			setChannel, err = s.State.Channel(chStr)
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

		err = b.db.UpdateGuild(g.ID, gc)
		if err != nil {
			_, _ = s.ChannelMessageSend(ch.ID, "Could not update config ")
			b.log.Error("failed to update guild config", zap.Error(err))
			return
		}

		_, _ = s.ChannelMessageSend(ch.ID, "Updated config")
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
		_, _ = s.ChannelMessageSend(ch.ID, text.String())
	}
}

func (b *Bot) messageDeleteHandler(s *discordgo.Session, m *discordgo.MessageDelete) {

	gc, err := b.db.GetGuild(m.GuildID)
	if err != nil || gc.MsgDeleteLog == "" {
		return
	}

	msg, err := b.store.GetMessage(fmt.Sprintf("%v:%v:%v", m.GuildID, m.ChannelID, m.ID))
	if err != nil {
		return
	}
	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		b.log.Info("error", zap.Error(err))
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

	gc, err := b.db.GetGuild(m.GuildID)
	if err != nil || gc.MsgDeleteLog == "" {
		return
	}

	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		b.log.Info("error", zap.Error(err))
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

	gc, err := b.db.GetGuild(m.GuildID)
	if err != nil || gc.MsgEditLog == "" {
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

*/
