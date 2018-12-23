package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ninedraft/simplepaste"

	"github.com/bwmarrin/discordgo"
)

type Config struct {
	OwnerID   string `json:"OwnerID"`
	Token     string `json:"Token"`
	PBToken   string `json:"PBToken"`
	MsgEdit   string `json:"MsgEdit"`
	MsgDelete string `json:"MsgDelete"`
	Ban       string `json:"Ban"`
	Unban     string `json:"Unban"`
	Join      string `json:"Join"`
	Leave     string `json:"Leave"`
}

type discMessage struct {
	message    *discordgo.Message
	attachment []byte
}

var (
	config       Config
	messageCache = make(map[string]*discMessage)
	memberCache  = make(map[string]*discordgo.Member)
	api          *simplepaste.API
)

const (
	dColorRed    = 13107200
	dColorOrange = 15761746
	dColorLBlue  = 6410733
	dColorGreen  = 51200
	dColorWhite  = 16777215
)

func main() {
	file, e := ioutil.ReadFile("./config.json")
	if e != nil {
		fmt.Printf("Config file not found.")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
		return
	}

	json.Unmarshal(file, &config)

	token := config.Token

	client, err := discordgo.New("Bot " + token)

	if err != nil {
		fmt.Println(err)
		return
	}

	api = simplepaste.NewAPI(config.PBToken)

	addHandlers(client)

	err = client.Open()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	client.Close()
}

func addHandlers(s *discordgo.Session) {
	s.AddHandler(GuildAvailableHandler)
	go s.AddHandler(GuildMemberUpdateHandler)
	go s.AddHandler(MessageUpdateHandler)

	if config.Join != "" {
		go s.AddHandler(MemberJoinedHandler)
	}

	if config.Leave != "" {
		go s.AddHandler(MemberLeaveHandler)
	}

	if config.Ban != "" {
		go s.AddHandler(MemberBannedHandler)
	}

	if config.Unban != "" {
		go s.AddHandler(MemberUnbannedHandler)
	}

	if config.MsgDelete != "" {
		go s.AddHandler(MessageDeleteHandler)
		go s.AddHandler(MessageDeleteBulkHandler)
	}

	go s.AddHandler(MessageCreateHandler)
	go s.AddHandler(ReadyHandler)
	go s.AddHandler(DisconnectHandler)
}

func GuildAvailableHandler(s *discordgo.Session, g *discordgo.GuildCreate) {
	for _, mem := range g.Members {
		memberCache[g.ID+mem.User.ID] = mem
	}
}
func GuildMemberUpdateHandler(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {
	if _, ok := memberCache[m.GuildID+m.User.ID]; ok {
		memberCache[m.GuildID+m.User.ID] = m.Member
	}
}
func GuildUnavailableHandler(s *discordgo.Session, g *discordgo.GuildDelete) {
	for _, mem := range g.Members {
		memberCache[g.ID+mem.User.ID] = mem
	}
}

func MemberJoinedHandler(s *discordgo.Session, m *discordgo.GuildMemberAdd) {

	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		return
	}

	if _, ok := memberCache[m.GuildID+m.User.ID]; !ok {
		memberCache[m.GuildID+m.User.ID] = m.Member
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
func MemberLeaveHandler(s *discordgo.Session, m *discordgo.GuildMemberRemove) {

	gamer := []string{}

	if mem, ok := memberCache[m.GuildID+m.User.ID]; ok {

		g, err := s.State.Guild(m.GuildID)
		if err != nil {
			return
		}

		for _, r := range mem.Roles {
			gamer = append(gamer, fmt.Sprintf("<@&%v>", r))
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

		if len(gamer) < 1 {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  "Roles",
				Value: "None",
			})
		} else if len(gamer) < 10 {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  "Roles",
				Value: strings.Join(gamer, ", "),
			})
		} else {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  "Roles",
				Value: fmt.Sprintf("%v and %v more", strings.Join(gamer[0:9], ", "), len(gamer)-9),
			})
		}

		_, err = s.ChannelMessageSendEmbed(config.Leave, &embed)
		if err != nil {
			fmt.Println("LEAVE LOG ERROR", err)
		}

		delete(memberCache, m.GuildID+m.User.ID)
	}
}

func MemberBannedHandler(s *discordgo.Session, m *discordgo.GuildBanAdd) {

	g, err := s.State.Guild(m.GuildID)
	if err != nil {
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

	if _, ok := memberCache[m.GuildID+m.User.ID]; ok {

		text := ""
		msgCount := 0

		for _, mc := range messageCache {
			if mc.message.Author.ID == m.User.ID {

				ch, err := s.State.Channel(mc.message.ChannelID)
				if err != nil {
					continue
				}

				if len(mc.attachment) > 0 {
					text += fmt.Sprintf("\nUser: %v (%v)\nChannel: %v (%v)\nContent: %v\nMessage had attachment\n", mc.message.Author.String(), mc.message.Author.ID, ch.Name, ch.ID, mc.message.Content)
				} else {
					text += fmt.Sprintf("\nUser: %v (%v)\nChannel: %v (%v)\nContent: %v\n", mc.message.Author.String(), mc.message.Author.ID, ch.Name, ch.ID, mc.message.Content)
				}
				msgCount++
			}
		}

		if msgCount > 0 {

			paste := simplepaste.NewPaste(fmt.Sprintf("24h ban log for %v (%v) - %v", m.User.String(), m.User.ID, ts.Format(time.RFC1123)), text)

			paste.ExpireDate = simplepaste.Never
			paste.Privacy = simplepaste.Unlisted

			link, err := api.SendPaste(paste)
			if err != nil {
				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
					Name:  "24h user log",
					Value: "Error getting pastebin link",
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
	} else {
		embed.Title += " - Hackban"
	}

	_, err = s.ChannelMessageSendEmbed(config.Ban, &embed)
	if err != nil {
		fmt.Println("BAN LOG ERROR", err)
	}
}

func MemberUnbannedHandler(s *discordgo.Session, m *discordgo.GuildBanRemove) {

	g, err := s.State.Guild(m.GuildID)
	if err != nil {
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

	_, err = s.ChannelMessageSendEmbed(config.Unban, &embed)
	if err != nil {
		fmt.Println("UNBAN LOG ERROR", err)
	}
}

func MessageUpdateHandler(s *discordgo.Session, m *discordgo.MessageUpdate) {

	if oldm, ok := messageCache[m.ID]; ok {

		g, err := s.State.Guild(m.GuildID)
		if err != nil {
			return
		}

		oldmsg := oldm.message

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
		_, err = s.ChannelMessageSendEmbed(config.MsgEdit, &embed)
		if err != nil {
			fmt.Println("EDIT LOG ERROR", err)
		}
	}
}

func MessageDeleteHandler(s *discordgo.Session, m *discordgo.MessageDelete) {

	if msg, ok := messageCache[m.ID]; ok {
		g, err := s.State.Guild(m.GuildID)
		if err != nil {
			return
		}

		msgo := msg.message

		embed := discordgo.MessageEmbed{
			Color: dColorWhite,
			Title: "Message deleted",
			Fields: []*discordgo.MessageEmbedField{
				&discordgo.MessageEmbedField{
					Name:   "User",
					Value:  fmt.Sprintf("%v\n%v\n%v", msgo.Author.Mention(), msgo.Author.String(), msgo.Author.ID),
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

		if msgo.Content != "" {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  "Content",
				Value: msgo.Content,
			})
		} else {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  "Content",
				Value: "No content",
			})
		}
		if len(msgo.Attachments) > 0 {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  "Total attachments",
				Value: fmt.Sprint(len(msgo.Attachments)),
			})
		}

		_, err = s.ChannelMessageSendEmbed(config.MsgDelete, &embed)
		if err != nil {
			fmt.Println("DELETE LOG ERROR", err)
		}
		if len(msgo.Attachments) > 0 {
			//ext := strings.Split()
			s.ChannelFileSendWithMessage(config.MsgDelete, fmt.Sprintf("File attached to message ID: %v", m.ID), msgo.Attachments[0].Filename, bytes.NewReader(msg.attachment))
		}
	}
}

func MessageDeleteBulkHandler(s *discordgo.Session, m *discordgo.MessageDeleteBulk) {

	g, err := s.State.Guild(m.GuildID)
	if err != nil {
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

	text := ""

	for _, mc := range m.Messages {
		if msg, ok := messageCache[mc]; ok {
			if len(msg.attachment) > 0 {
				text += fmt.Sprintf("\nUser: %v (%v)\nContent: %v\nMessage had attachment\n", msg.message.Author.String(), msg.message.Author.ID, msg.message.Content)
			} else {
				text += fmt.Sprintf("\nUser: %v (%v)\nContent: %v\n", msg.message.Author.String(), msg.message.Author.ID, msg.message.Content)
			}
		}
	}

	paste := simplepaste.NewPaste(fmt.Sprintf("%v - %v", m.ChannelID, ts.Format(time.RFC1123)), text)

	paste.ExpireDate = simplepaste.Never
	paste.Privacy = simplepaste.Unlisted

	link, err := api.SendPaste(paste)
	if err != nil {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Pastebin log link",
			Value: "Error getting pastebin link",
		})
	} else {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Pastebin log link",
			Value: link,
		})
	}

	_, err = s.ChannelMessageSendEmbed(config.MsgDelete, &embed)
	if err != nil {
		fmt.Println("BULK DELETE LOG ERROR", err)
	}
}

func MessageCreateHandler(s *discordgo.Session, m *discordgo.MessageCreate) {

	if m.Author.Bot {
		return
	}

	ch, err := s.State.Channel(m.ChannelID)
	if err != nil {
		fmt.Println("GUILD ERROR", err)
		return
	}

	if ch.Type != discordgo.ChannelTypeGuildText {
		return
	}

	g, err := s.State.Guild(ch.GuildID)
	if err != nil {
		fmt.Println("CHANNEL ERROR", err)
		return
	}

	fmt.Println(fmt.Sprintf("%v - %v - %v: %v", g.Name, ch.Name, m.Author.String(), m.Content))

	dmsg := &discMessage{
		message:    m.Message,
		attachment: []byte{},
	}

	if len(m.Attachments) > 0 {

		res, _ := http.Get(m.Attachments[0].URL)
		if err != nil {
			return
		}

		d, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return
		}

		dmsg.attachment = d
	}

	messageCache[m.ID] = dmsg

	if m.Content == "fl.len" {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprint(len(messageCache)))
	} else if m.Content == "fl.mlen" {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprint(len(memberCache)))
	} else if m.Author.ID == config.OwnerID {
		if m.Content == "fl.clear" {
			messageCache = map[string]*discMessage{}
		}
	}

	go func() {
		cleartime := time.After(24 * time.Hour)

		select {
		case <-cleartime:
			delete(messageCache, m.ID)
		}
	}()
}
func ReadyHandler(s *discordgo.Session, r *discordgo.Ready) {
	fmt.Println(fmt.Sprintf("Logged in as %v.", r.User.String()))
}
func DisconnectHandler(s *discordgo.Session, d *discordgo.Disconnect) {
	fmt.Println("DISCONNECTED AT ", time.Now().Format(time.RFC1123))
}
