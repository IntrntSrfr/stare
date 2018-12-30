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
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/dgraph-io/badger/options"

	"github.com/dgraph-io/badger"

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
	Message    *discordgo.Message
	Attachment [][]byte
}

var (
	config Config
	api    *simplepaste.API
	memDB  *badger.DB
	msgDB  *badger.DB
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

	msgPath, _ := filepath.Abs("../functional-logger/tmp/msg")
	memPath, _ := filepath.Abs("../functional-logger/tmp/mem")

	opts := badger.DefaultOptions
	opts.Dir = msgPath
	opts.ValueDir = msgPath
	opts.ValueLogLoadingMode = options.FileIO
	opts.ReadOnly = false

	db, err := badger.Open(opts)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	msgDB = db

	opts = badger.DefaultOptions
	opts.Dir = memPath
	opts.ValueDir = memPath
	opts.ValueLogLoadingMode = options.FileIO
	opts.ReadOnly = false

	db, err = badger.Open(opts)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	memDB = db

	token := config.Token

	client, err := discordgo.New("Bot " + token)

	if err != nil {
		fmt.Println(err)
		return
	}

	api = simplepaste.NewAPI(config.PBToken)
	removeOldMessages()

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
		//go s.AddHandler(MessageDeleteBulkHandler)
	}

	go s.AddHandler(MessageCreateHandler)
	go s.AddHandler(ReadyHandler)
	go s.AddHandler(DisconnectHandler)
}

func GuildAvailableHandler(s *discordgo.Session, g *discordgo.GuildCreate) {
	for _, mem := range g.Members {
		setMember(mem)
	}
}
func GuildMemberUpdateHandler(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {
	setMember(m.Member)
}

func MemberJoinedHandler(s *discordgo.Session, m *discordgo.GuildMemberAdd) {

	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		return
	}

	if _, err := getMember(m.GuildID + m.User.ID); err != nil {
		setMember(m.Member)
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

	if mem, err := getMember(m.GuildID + m.User.ID); err == nil {

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

		txn := memDB.NewTransaction(true)
		defer txn.Discard()
		txn.Delete([]byte(m.GuildID + m.User.ID))
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

	if _, err := getMember(m.GuildID + m.User.ID); err == nil {

		text := ""
		msgCount := 0

		txn := msgDB.NewTransaction(false)
		defer txn.Discard()
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		messageCache := []*discMessage{}

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			err := item.Value(func(v []byte) error {
				dmsg := &discMessage{}
				err := json.Unmarshal(v, &dmsg)
				if err != nil {
					return err
				}

				if dmsg.Message.Author.ID == m.User.ID {
					messageCache = append(messageCache, dmsg)
				}

				return nil
			})
			if err != nil {
				continue
			}
		}

		for _, mc := range messageCache {
			if mc.Message.Author.ID == m.User.ID {

				ch, err := s.State.Channel(mc.Message.ChannelID)
				if err != nil {
					continue
				}

				if len(mc.Attachment) > 0 {
					text += fmt.Sprintf("\nUser: %v (%v)\nChannel: %v (%v)\nContent: %v\nMessage had attachment\n", mc.Message.Author.String(), mc.Message.Author.ID, ch.Name, ch.ID, mc.Message.Content)
				} else {
					text += fmt.Sprintf("\nUser: %v (%v)\nChannel: %v (%v)\nContent: %v\n", mc.Message.Author.String(), mc.Message.Author.ID, ch.Name, ch.ID, mc.Message.Content)
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

	oldm, err := getMessage(m.ID)
	if err != nil {
		return
	}

	g, err := s.State.Guild(m.GuildID)
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
	_, err = s.ChannelMessageSendEmbed(config.MsgEdit, &embed)
	if err != nil {
		fmt.Println("EDIT LOG ERROR", err)
	}
}

func MessageDeleteHandler(s *discordgo.Session, m *discordgo.MessageDelete) {

	msg, err := getMessage(m.ID)
	if err != nil {
		fmt.Println(err)
		return
	}

	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		return
	}

	msgo := msg.Message

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

	for k := range msgo.Attachments {
		s.ChannelFileSendWithMessage(config.MsgDelete, fmt.Sprintf("File attached to message ID: %v", m.ID), msgo.Attachments[k].Filename, bytes.NewReader(msg.Attachment[k]))
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

	messageCache := []*discMessage{}

	for _, val := range m.Messages {
		dmsg, err := getMessage(val)
		if err != nil {
			continue
		}

		messageCache = append(messageCache, dmsg)
	}

	text := ""

	for _, msg := range messageCache {

		if len(msg.Message.Attachments) > 0 {
			text += fmt.Sprintf("\nUser: %v (%v)\nContent: %v\nMessage had attachment\n", msg.Message.Author.String(), msg.Message.Author.ID, msg.Message.Content)
		} else {
			text += fmt.Sprintf("\nUser: %v (%v)\nContent: %v\n", msg.Message.Author.String(), msg.Message.Author.ID, msg.Message.Content)
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

	var err error

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

	err = setMessage(m.Message)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(fmt.Sprintf("%v - %v - %v: %v", g.Name, ch.Name, m.Author.String(), m.Content))
}

func ReadyHandler(s *discordgo.Session, r *discordgo.Ready) {
	fmt.Println(fmt.Sprintf("Logged in as %v.", r.User.String()))
}
func DisconnectHandler(s *discordgo.Session, d *discordgo.Disconnect) {
	fmt.Println("DISCONNECTED AT ", time.Now().Format(time.RFC1123))
}

func setMessage(m *discordgo.Message) error {
	err := msgDB.Update(func(txn *badger.Txn) error {
		dmsg := &discMessage{
			Message:    m,
			Attachment: [][]byte{},
		}

		for _, val := range m.Attachments {
			res, err := http.Get(val.URL)
			if err != nil {
				return err
			}

			d, err := ioutil.ReadAll(res.Body)
			if err != nil {
				return err
			}

			dmsg.Attachment = append(dmsg.Attachment, d)
		}

		b, err := json.Marshal(dmsg)
		if err != nil {
			return err
		}

		err = txn.Set([]byte(m.ID), b)

		return err
	})
	if err != nil {
		fmt.Println(err)
		return err
	}

	go func() {
		timer := time.After(24 * time.Hour)

		select {
		case <-timer:
			txn := msgDB.NewTransaction(true)
			defer txn.Discard()
			err := txn.Delete([]byte(m.ID))
			if err != nil {
				fmt.Println(err)
			}
		}
	}()
	return nil
}

func getMessage(ID string) (*discMessage, error) {
	var (
		err     error
		valCopy []byte
	)
	err = msgDB.View(func(txn *badger.Txn) error {

		item, err := txn.Get([]byte(ID))
		if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			valCopy = append([]byte{}, val...)
			return nil
		})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	msg := discMessage{}
	err = json.Unmarshal(valCopy, &msg)

	return &msg, err
}

func setMember(m *discordgo.Member) error {
	err := memDB.Update(func(txn *badger.Txn) error {
		b, err := json.Marshal(m)
		if err != nil {
			return err
		}

		err = txn.Set([]byte(m.GuildID+m.User.ID), b)
		return err
	})
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func getMember(ID string) (*discordgo.Member, error) {
	var (
		err     error
		valCopy []byte
	)
	err = memDB.View(func(txn *badger.Txn) error {

		item, err := txn.Get([]byte(ID))
		if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			valCopy = append([]byte{}, val...)
			return nil
		})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	msg := discordgo.Member{}
	err = json.Unmarshal(valCopy, &msg)

	return &msg, err
}

func removeOldMessages() {
	msgDB.Update(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := it.Item().KeyCopy(nil)
			err := item.Value(func(v []byte) error {
				dmsg := &discMessage{}
				err := json.Unmarshal(v, &dmsg)
				if err != nil {
					return err
				}

				id, err := strconv.ParseInt(dmsg.Message.ID, 0, 63)
				if err != nil {
					return err
				}

				id = ((id >> 22) + 1420070400000) / 1000
				dur := time.Since(time.Unix(int64(id), 0))

				if dur > time.Hour*24 {
					err := txn.Delete(k)
					if err != nil {
						fmt.Println(err)
					} else {
						fmt.Println("Deleted", string(k))
					}
				}

				return nil
			})
			if err != nil {
				continue
			}
		}
		return nil
	})
}
