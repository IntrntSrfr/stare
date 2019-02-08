package loggerbot

import (
	"bytes"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/intrntsrfr/functional-logger/owo"

	"go.uber.org/zap"

	"github.com/bwmarrin/discordgo"
	"github.com/intrntsrfr/functional-logger/loggerdb"
)

type Bot struct {
	loggerDB  *loggerdb.DB
	logger    *zap.Logger
	client    *discordgo.Session
	config    *Config
	owo       *owo.OWOClient
	starttime time.Time
}

func NewLoggerBot(Config *Config, LoggerDB *loggerdb.DB, Log *zap.Logger) (*Bot, error) {

	client, err := discordgo.New("Bot " + Config.Token)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	Log.Info("created discord client")

	owoCl := owo.NewOWOClient(Config.OwoAPIKey)
	Log.Info("created owo client")

	return &Bot{
		client:    client,
		config:    Config,
		logger:    Log,
		loggerDB:  LoggerDB,
		owo:       owoCl,
		starttime: time.Now(),
	}, nil

}
func (b *Bot) Close() {
	b.client.Close()
}

func (b *Bot) Run() error {
	b.loggerDB.LoadTotal()

	b.addHandlers()
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	return b.client.Open()
}

func (b *Bot) addHandlers() {
	b.client.AddHandler(b.guildCreateHandler)
	b.client.AddHandler(b.guildMemberUpdateHandler)
	b.client.AddHandler(b.guildMemberAddHandler)
	b.client.AddHandler(b.guildMemberRemoveHandler)
	b.client.AddHandler(b.guildBanAddHandler)
	b.client.AddHandler(b.guildBanRemoveHandler)
	b.client.AddHandler(b.messageCreateHandler)
	b.client.AddHandler(b.messageUpdateHandler)
	b.client.AddHandler(b.messageDeleteHandler)
	b.client.AddHandler(b.messageDeleteBulkHandler)
	b.client.AddHandler(b.readyHandler)
	b.client.AddHandler(b.disconnectHandler)
}

func (b *Bot) guildCreateHandler(s *discordgo.Session, g *discordgo.GuildCreate) {

	b.logger.Info("EVENT: GUILD CREATE")

	go func() {
		for _, mem := range g.Members {
			/*
				go func(m *discordgo.Member) {
					err = LoadMember(m)
				}(mem)
			*/

			err := b.loggerDB.SetMember(mem, 1)
			if err != nil {
				b.logger.Info("error", zap.Error(err))
				continue
			}
		}
		b.logger.Info(fmt.Sprintf("LOADED %v", g.Name))
		fmt.Println("loaded", g.Name)
	}()
}

func (b *Bot) guildUnavailableHandler(s *discordgo.Session, g *discordgo.GuildDelete) {

}

func (b *Bot) guildMemberUpdateHandler(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {

	err := b.loggerDB.SetMember(m.Member, 0)
	if err != nil {
		fmt.Println(err)
		b.logger.Info("error", zap.Error(err))
		return
	}
}

func (b *Bot) guildMemberAddHandler(s *discordgo.Session, m *discordgo.GuildMemberAdd) {

	err := b.loggerDB.SetMember(m.Member, 1)
	if err != nil {
		fmt.Println(err)
		b.logger.Info("error", zap.Error(err))
		return
	}
	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		b.logger.Info("error", zap.Error(err))
		return
	}

	id, err := strconv.ParseInt(m.User.ID, 0, 63)
	if err != nil {
		b.logger.Info("error", zap.Error(err))
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

	_, err = s.ChannelMessageSendEmbed(b.config.Join, &embed)
	if err != nil {
		b.logger.Info("error", zap.Error(err))
		fmt.Println("JOIN LOG ERROR", err)
	}
}

func (b *Bot) guildMemberRemoveHandler(s *discordgo.Session, m *discordgo.GuildMemberRemove) {
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

	_, err = s.ChannelMessageSendEmbed(b.config.Leave, &embed)
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

func (b *Bot) guildBanAddHandler(s *discordgo.Session, m *discordgo.GuildBanAdd) {
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

			link, err := b.owo.Upload(text.String())
			if err != nil {
				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
					Name:  "24h user log",
					Value: "Error getting link",
				})
			} else {
				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
					Name:  "24h user log",
					Value: link,
				})
			}
		} else {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  "24h user log",
				Value: "No history.",
			})
		}
	}

	_, err = s.ChannelMessageSendEmbed(b.config.Ban, &embed)
	if err != nil {
		b.logger.Info("error", zap.Error(err))
		fmt.Println("BAN LOG ERROR", err)
	}
}

func (b *Bot) guildBanRemoveHandler(s *discordgo.Session, m *discordgo.GuildBanRemove) {
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

	_, err = s.ChannelMessageSendEmbed(b.config.Unban, &embed)
	if err != nil {
		b.logger.Info("error", zap.Error(err))
		fmt.Println("UNBAN LOG ERROR", err)
	}
}

func (b *Bot) messageUpdateHandler(s *discordgo.Session, m *discordgo.MessageUpdate) {
	// This means it was an image update and not an actual edit
	if m.Message.Content == "" {
		return
	}

	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		b.logger.Info("error", zap.Error(err))
		return
	}

	oldm, err := b.loggerDB.GetMessage(fmt.Sprintf("%v:%v:%v", m.GuildID, m.ChannelID, m.ID))
	if err != nil {
		return
	}

	oldmsg := oldm.Message

	if oldmsg.Content == m.Content {
		return
	}

	embed := discordgo.MessageEmbed{
		Color: dColorLBlue,
		Title: "Message edited",
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
				Name:   "User",
				Value:  fmt.Sprintf("%v\n%v\n%v", oldmsg.Author.Mention(), oldmsg.Author.String(), oldmsg.Author.ID),
				Inline: true,
			},
			&discordgo.MessageEmbedField{
				Name:   "Message ID",
				Value:  m.ID,
				Inline: true,
			},
			&discordgo.MessageEmbedField{
				Name:  "Channel",
				Value: fmt.Sprintf("<#%v> (%v)", m.ChannelID, m.ChannelID),
			},
			&discordgo.MessageEmbedField{
				Name:  "Old content",
				Value: oldmsg.Content,
			},
			&discordgo.MessageEmbedField{
				Name:  "New content",
				Value: m.Content,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			IconURL: discordgo.EndpointGuildIcon(g.ID, g.Icon),
			Text:    g.Name,
		},
	}
	_, err = s.ChannelMessageSendEmbed(b.config.MsgEdit, &embed)
	if err != nil {
		b.logger.Info("error", zap.Error(err))
		fmt.Println("EDIT LOG ERROR", err)
	}

	oldm.Message.Content = m.Content

	err = b.loggerDB.SetMessage(oldm.Message)
	if err != nil {
		b.logger.Info("error", zap.Error(err))
		fmt.Println("ERROR")
		return
	}
}

func (b *Bot) messageDeleteHandler(s *discordgo.Session, m *discordgo.MessageDelete) {

	msg, err := b.loggerDB.GetMessage(fmt.Sprintf("%v:%v:%v", m.GuildID, m.ChannelID, m.ID))
	if err != nil {
		//fmt.Println(err)
		return
	}
	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		b.logger.Info("error", zap.Error(err))
		return
	}

	embed := discordgo.MessageEmbed{
		Color: dColorWhite,
		Title: "Message deleted",
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
				Name:   "User",
				Value:  fmt.Sprintf("%v\n%v\n%v", msg.Message.Author.Mention(), msg.Message.Author.String(), msg.Message.Author.ID),
				Inline: true,
			},
			&discordgo.MessageEmbedField{
				Name:   "Message ID",
				Value:  m.ID,
				Inline: true,
			},
			&discordgo.MessageEmbedField{
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
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Content",
			Value: msg.Message.Content,
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

	_, err = s.ChannelMessageSendEmbed(b.config.MsgDelete, &embed)
	if err != nil {
		b.logger.Info("error", zap.Error(err))
		fmt.Println("DELETE LOG ERROR", err)
	}
	if len(msg.Message.Attachments) > 0 {
		send, err := s.ChannelMessageSend(b.config.MsgDelete, "Trying to get attachments..")
		if err != nil {
			b.logger.Info("error", zap.Error(err))
			fmt.Println("DELETE LOG SEND ERROR", err)
			return
		}
		data := &discordgo.MessageSend{
			Content: fmt.Sprintf("File(s) attached to message ID:%v", m.ID),
		}

		for k, img := range msg.Attachments {
			f := &discordgo.File{
				Name:   msg.Message.Attachments[k].Filename,
				Reader: bytes.NewReader(img),
			}
			data.Files = append(data.Files, f)
		}

		_, err = s.ChannelMessageSendComplex(b.config.MsgDelete, data)
		if err != nil {
			b.logger.Info("error", zap.Error(err))
			s.ChannelMessageEdit(send.ChannelID, send.ID, "Error getting attachments")
			fmt.Println("DELETE LOG ERROR", err)
		} else {
			s.ChannelMessageDelete(send.ChannelID, send.ID)
		}
	}
}

func (b *Bot) messageDeleteBulkHandler(s *discordgo.Session, m *discordgo.MessageDeleteBulk) {
	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		b.logger.Info("error", zap.Error(err))
		return
	}
	ts := time.Now()

	embed := discordgo.MessageEmbed{
		Color: dColorWhite,
		Title: fmt.Sprintf("Bulk message delete - (%v) messages deleted", len(m.Messages)),
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
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
	deletedmsgs := []*loggerdb.DMsg{}
	for _, msgid := range m.Messages {
		delmsg, err := b.loggerDB.GetMessage(fmt.Sprintf("%v:%v:%v", m.GuildID, m.ChannelID, msgid))
		if err != nil {
			b.logger.Info("error", zap.Error(err))
			continue
		}
		deletedmsgs = append(deletedmsgs, delmsg)
	}

	sort.Sort(loggerdb.ByID(deletedmsgs))

	text := strings.Builder{}
	text.WriteString(fmt.Sprintf("%v - %v\n\n\n", m.ChannelID, ts.Format(time.RFC1123)))

	for _, msg := range deletedmsgs {
		if len(msg.Attachments) > 0 {
			text.WriteString(fmt.Sprintf("\nUser: %v (%v)\nContent: %v\nMessage had attachment\n", msg.Message.Author.String(), msg.Message.Author.ID, msg.Message.Content))
		} else {
			text.WriteString(fmt.Sprintf("\nUser: %v (%v)\nContent: %v\n", msg.Message.Author.String(), msg.Message.Author.ID, msg.Message.Content))
		}
	}

	if b.config.OwoAPIKey != "" {

		res, err := b.owo.Upload(text.String())

		if err != nil {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  "Logged messages:",
				Value: "Error getting link",
			})
		} else {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  "Logged messages:",
				Value: res,
			})
		}
		_, err = s.ChannelMessageSendEmbed(b.config.MsgDelete, &embed)
		if err != nil {
			fmt.Println("BULK DELETE LOG ERROR", err)
		}
	} else {
		jeff := bytes.Buffer{}
		jeff.WriteString(text.String())

		msg, err := s.ChannelMessageSendEmbed(b.config.MsgDelete, &embed)
		if err != nil {
			fmt.Println("BULK DELETE LOG ERROR", err)
		}

		s.ChannelFileSendWithMessage(b.config.MsgDelete, fmt.Sprintf("Log file for delete log message ID %v:", msg.ID), "deletelog_"+m.ChannelID+".txt", &jeff)
	}
}

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
}

func (b *Bot) readyHandler(s *discordgo.Session, r *discordgo.Ready) {
	fmt.Println(fmt.Sprintf("Logged in as %v.", r.User.String()))
}

func (b *Bot) disconnectHandler(s *discordgo.Session, d *discordgo.Disconnect) {
	atomic.StoreInt64(&b.loggerDB.TotalMembers, 0)
	fmt.Println("DISCONNECTED AT ", time.Now().Format(time.RFC1123))
}
