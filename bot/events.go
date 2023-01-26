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

func disconnectHandler(_ *Context, _ *discordgo.Disconnect) {
}

func guildBanAddHandler(c *Context, m *discordgo.GuildBanAdd) {
	g, err := c.s.State.Guild(m.GuildID)
	if err != nil {
		c.b.log.Error("failed to fetch guild", zap.Error(err))
		return
	}

	reply := &discordgo.MessageSend{Embed: NewLogEmbed(GuildBanType, g)}
	reply.Embed = SetEmbedThumbnail(reply.Embed, m.User.AvatarURL("256"))
	reply.Embed = AddEmbedField(reply.Embed, "User", fmt.Sprintf("%v\n%v", m.User.Mention(), m.User.String()), false)
	reply.Embed = AddEmbedField(reply.Embed, "ID", m.User.ID, false)

	if _, err = c.b.store.GetMember(m.GuildID, m.User.ID); err != nil {
		if err != badger.ErrKeyNotFound {
			return
		}
		// if user is not found, aka they were never in the server
		reply.Embed.Title += " - Hackban"

		_, err = c.s.ChannelMessageSendEmbed(c.gc.BanLog, reply.Embed)
		if err != nil {
			c.b.log.Info("failed to send log message", zap.Error(err))
		}
		return
	}

	var files []*discordgo.File
	for _, ch := range g.Channels {
		chLog, err := c.b.store.GetMessageLog(g.ID, ch.ID, m.User.ID)
		if err != nil || len(chLog) == 0 {
			continue
		}
		sort.Sort(kvstore.ByID(chLog))

		buf := &bytes.Buffer{}
		buf.WriteString(fmt.Sprintf("Log for user: %v (%v); channel: %v (%v)\n", m.User, m.User.ID, ch.Name, ch.ID))
		for _, msg := range chLog {
			str := fmt.Sprintf("\nContent: %v\n", msg.Message.Content)
			if len(msg.Attachments) > 0 {
				str += "Message had attachment\n"
			}
			buf.WriteString(str)
		}

		reply = AddMessageFile(reply, fmt.Sprintf("%v_%v_%v.txt", g.ID, ch.ID, m.User.ID), buf.Bytes())
	}

	if len(files) == 0 {
		reply.Embed = AddEmbedField(reply.Embed, "24h user log", "No history", false)
		_, _ = c.s.ChannelMessageSendEmbed(c.gc.BanLog, reply.Embed)
		return
	}

	_, _ = c.s.ChannelMessageSendComplex(c.gc.BanLog, reply)
}

func guildBanRemoveHandler(c *Context, m *discordgo.GuildBanRemove) {
	g, err := c.s.State.Guild(m.GuildID)
	if err != nil {
		return
	}

	embed := NewLogEmbed(GuildUnbanType, g)
	embed = SetEmbedThumbnail(embed, m.User.AvatarURL("256"))
	embed = AddEmbedField(embed, "User", fmt.Sprintf("%v\n%v", m.User.Mention(), m.User.String()), false)
	embed = AddEmbedField(embed, "ID", m.User.ID, false)
	_, _ = c.s.ChannelMessageSendEmbed(c.gc.UnbanLog, embed)
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
	embed := NewLogEmbed(GuildJoinType, g)
	embed = SetEmbedThumbnail(embed, m.User.AvatarURL("256"))
	embed = AddEmbedField(embed, "User", fmt.Sprintf("%v\n%v (%v)", m.User.Mention(), m.User.String(), m.User.ID), false)
	embed = AddEmbedField(embed, "Creation date", fmt.Sprintf("<t:%v:R>", ts.Unix()), false)
	_, _ = c.s.ChannelMessageSendEmbed(c.gc.JoinLog, embed)
}

func guildMemberRemoveHandler(c *Context, m *discordgo.GuildMemberRemove) {
	g, err := c.s.State.Guild(m.GuildID)
	if err != nil {
		return
	}

	mem, err := c.b.store.GetMember(m.GuildID, m.User.ID)
	if err != nil {
		return
	}

	embed := NewLogEmbed(GuildLeaveType, g)
	embed = SetEmbedThumbnail(embed, m.User.AvatarURL("256"))
	embed = AddEmbedField(embed, "User", fmt.Sprintf("%v\n%v", m.User.Mention(), m.User.String()), true)
	embed = AddEmbedField(embed, "ID", m.User.ID, true)

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

	_, _ = c.s.ChannelMessageSendEmbed(c.gc.LeaveLog, embed)

	err = c.b.store.DeleteMember(m.GuildID, m.User.ID)
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
	if m.Content == "" {
		return
	}

	if strings.HasPrefix(m.Content, "fl.info") {
		_, _ = c.s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
			Title: "Info",
			Color: Blue,
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

	args := strings.Fields(m.Content)
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

		if uperms&(discordgo.PermissionAdministrator|discordgo.PermissionAll) == 0 {
			_, _ = c.s.ChannelMessageSend(ch.ID, "This is admin only, sorry!")
			return
		}

		setChannel := ch
		if len(args) >= 3 {
			chStr := TrimChannelString(args[2])
			setChannel, err = c.s.State.Channel(chStr)
			if err != nil || setChannel.GuildID != g.ID {
				c.b.log.Debug("failed to get guild", zap.Error(err))
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

func messageDeleteHandler(c *Context, m *discordgo.MessageDelete) {
	msg, err := c.b.store.GetMessage(m.GuildID, m.ChannelID, m.ID)
	if err != nil {
		return
	}

	g, err := c.s.State.Guild(m.GuildID)
	if err != nil {
		return
	}

	reply := &discordgo.MessageSend{Embed: NewLogEmbed(MessageDeleteType, g)}
	reply.Embed = AddEmbedField(reply.Embed, "User", fmt.Sprintf("%v\n%v\n%v", msg.Message.Author.Mention(), msg.Message.Author.String(), msg.Message.Author.ID), true)
	reply.Embed = AddEmbedField(reply.Embed, "Message ID", m.ID, true)
	reply.Embed = AddEmbedField(reply.Embed, "Channel", fmt.Sprintf("<#%v> (%v)", m.ChannelID, m.ChannelID), false)

	contentStr := ""
	if msg.Message.Content == "" {
		contentStr = "No content"
	} else {
		str := msg.Message.Content
		if len(str) > 1024 {
			str = "Content too long, so it's put in the attached .txt file"
			reply = AddMessageFileString(reply, "content.txt", msg.Message.Content)
		}
		contentStr = str
	}
	reply.Embed = AddEmbedField(reply.Embed, "Content", contentStr, false)

	if len(msg.Attachments) > 0 {
		reply.Embed = AddEmbedField(reply.Embed, "Total fetched attachments", fmt.Sprint(len(msg.Attachments)), false)
		reply.Embed = AddEmbedField(reply.Embed, "Disclaimer", "It may only fetch attachments smaller than 10mb", false)
	}

	for _, a := range msg.Attachments {
		reply = AddMessageFile(reply, a.Filename, a.Data)
	}

	_, _ = c.s.ChannelMessageSendComplex(c.gc.MsgDeleteLog, reply)
}

func messageDeleteBulkHandler(c *Context, m *discordgo.MessageDeleteBulk) {
	g, err := c.d.Guild(m.GuildID)
	if err != nil {
		return
	}

	reply := &discordgo.MessageSend{Embed: NewLogEmbed(MessageDeleteBulkType, g)}
	reply.Embed.Title += fmt.Sprintf(" (%v) messages", len(m.Messages))
	reply.Embed = AddEmbedField(reply.Embed, "Channel", fmt.Sprintf("<#%v>", m.ChannelID), true)

	var messages []*kvstore.DiscordMessage
	for _, msgID := range m.Messages {
		msg, err := c.b.store.GetMessage(m.GuildID, m.ChannelID, msgID)
		if err != nil {
			continue
		}
		messages = append(messages, msg)
	}
	sort.Sort(kvstore.ByID(messages))

	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("%v - %v\n\n\n", m.ChannelID, time.Now().Format(time.RFC3339)))
	for _, msg := range messages {
		text := fmt.Sprintf("\nUser: %v (%v)\nContent: %v\n", msg.Message.Author.String(), msg.Message.Author.ID, msg.Message.Content)
		if len(msg.Attachments) > 0 {
			text += "Message had attachment\n"
		}
		builder.WriteString(text)
	}
	reply = AddMessageFileString(reply, fmt.Sprintf("deleted_%v.txt", m.ChannelID), builder.String())
	_, _ = c.s.ChannelMessageSendComplex(c.gc.MsgDeleteLog, reply)
}

func messageUpdateHandler(c *Context, m *discordgo.MessageUpdate) {
	// This means it was an image update and not an actual edit
	if m.Message.Content == "" {
		return
	}

	g, err := c.d.Guild(m.GuildID)
	if err != nil {
		c.b.log.Info("error", zap.Error(err))
		return
	}

	oldMsg, err := c.b.store.GetMessage(m.GuildID, m.ChannelID, m.ID)
	if err != nil {
		return
	}

	if oldMsg.Message.Content == m.Content {
		return
	}

	reply := &discordgo.MessageSend{Embed: NewLogEmbed(MessageUpdateType, g)}
	reply.Embed = AddEmbedField(reply.Embed, "User", fmt.Sprintf("%v\n%v\n%v", m.Author.Mention(), m.Author.String(), m.Author.ID), true)
	reply.Embed = AddEmbedField(reply.Embed, "Message ID", m.ID, true)
	reply.Embed = AddEmbedField(reply.Embed, "Channel", fmt.Sprintf("<#%v> (%v)", m.ChannelID, m.ChannelID), false)

	// check old content
	if len(oldMsg.Message.Content) > 1024 {
		reply = AddMessageFileString(reply, "old_content.txt", oldMsg.Message.Content)
	} else {
		reply.Embed = AddEmbedField(reply.Embed, "Old content", oldMsg.Message.Content, false)
	}

	// check new content
	if len(m.Content) > 1024 {
		reply = AddMessageFileString(reply, "new_content.txt", m.Content)
	} else {
		reply.Embed = AddEmbedField(reply.Embed, "New content", m.Content, false)
	}

	_, _ = c.s.ChannelMessageSendComplex(c.gc.MsgEditLog, reply)

	// I think this should be put in its own function and not at the end of this one lol
	oldMsg.Message.Content = m.Content
	err = c.b.store.SetMessage(oldMsg)
	if err != nil {
		c.b.log.Error("failed to update message", zap.Error(err))
		return
	}
}
