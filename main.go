package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/dgraph-io/badger"

	"github.com/bwmarrin/discordgo"
	_ "github.com/lib/pq"
)

var (
	config Config
	OWOC   *OWOClient
	memDB  *badger.DB
	msgDB  *badger.DB
	db     *sql.DB
	err    error
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
		fmt.Printf("Config file not found.\nPlease press enter.")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
		return
	}

	json.Unmarshal(file, &config)

	msgDB, err = NewMessageDB()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer msgDB.Close()
	memDB, err = NewMemberDB()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer memDB.Close()

	client, err := discordgo.New("Bot " + config.Token)
	if err != nil {
		fmt.Println(err)
		return
	}

	db, err = sql.Open("postgres", config.ConnectionString)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	OWOC = NewOWOClient(config.OWOApiKey)

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
	go s.AddHandler(GuildCreateHandler)
	go s.AddHandler(GuildMemberUpdateHandler)
	go s.AddHandler(GuildMemberAddHandler)
	go s.AddHandler(GuildMemberRemoveHandler)
	go s.AddHandler(GuildBanAddHandler)
	go s.AddHandler(GuildBanRemoveHandler)
	go s.AddHandler(MessageCreateHandler)
	go s.AddHandler(MessageUpdateHandler)
	go s.AddHandler(MessageDeleteHandler)
	go s.AddHandler(MessageDeleteBulkHandler)
	go s.AddHandler(ReadyHandler)
	go s.AddHandler(DisconnectHandler)
}

func GuildCreateHandler(s *discordgo.Session, g *discordgo.GuildCreate) {

	var count int

	row := db.QueryRow("SELECT COUNT(*) FROM discordguilds WHERE guildid = $1;", g.ID)

	err := row.Scan(&count)
	if err != nil {
		return
	}
	if count == 0 {
		_, err := db.Exec("INSERT INTO discordguilds(guildid, msgeditlog, msgdeletelog, banlog, unbanlog, joinlog, leavelog) VALUES($1, $2, $3, $4, $5, $6, $7);", g.ID, "", "", "", "", "", "")
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	for _, mem := range g.Members {
		err = LoadMember(mem)
		if err != nil {
			continue
		}
	}

	fmt.Println("loaded", g.Name)
}

func GuildUnavailableHandler(s *discordgo.Session, g *discordgo.GuildDelete) {

}

func GuildMemberUpdateHandler(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {
	err := LoadMember(m.Member)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func GuildMemberAddHandler(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	err := LoadMember(m.Member)
	if err != nil {
		fmt.Println(err)
		return
	}

	row := db.QueryRow("SELECT joinlog FROM discordguilds WHERE guildid=$1;", m.GuildID)
	dg := DiscordGuild{}
	err = row.Scan(&dg.JoinLog)
	if err != nil {
		fmt.Println(err)
		return
	}
	if dg.JoinLog == "" {
		return
	}
	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		return
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

	_, err = s.ChannelMessageSendEmbed(dg.JoinLog, &embed)
	if err != nil {
		fmt.Println("JOIN LOG ERROR", err)
	}
}

func GuildMemberRemoveHandler(s *discordgo.Session, m *discordgo.GuildMemberRemove) {
	row := db.QueryRow("SELECT leavelog FROM discordguilds WHERE guildid=$1;", m.GuildID)
	dg := DiscordGuild{}
	err := row.Scan(&dg.LeaveLog)
	if err != nil {
		fmt.Println(err)
		return
	}
	if dg.LeaveLog == "" {
		return
	}

	roles := []string{}

	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		return
	}

	mem, err := GetMember(fmt.Sprintf("%v:%v", m.GuildID, m.User.ID))

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

	_, err = s.ChannelMessageSendEmbed(dg.LeaveLog, &embed)
	if err != nil {
		fmt.Println("LEAVE LOG ERROR", err)
	}

	err = DeleteMember(fmt.Sprintf("%v:%v", m.GuildID, m.User.ID))
	if err != nil {
		fmt.Println(err)
		return
	}
}

func GuildBanAddHandler(s *discordgo.Session, m *discordgo.GuildBanAdd) {
	row := db.QueryRow("SELECT banlog FROM discordguilds WHERE guildid=$1;", m.GuildID)
	dg := DiscordGuild{}
	err := row.Scan(&dg.BanLog)
	if err != nil {
		fmt.Println(err)
		return
	}
	if dg.BanLog == "" {
		return
	}
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

	_, err = GetMember(fmt.Sprintf("%v:%v", m.GuildID, m.User.ID))
	if err != nil {
		/*
			d, _ := json.MarshalIndent(cmem, "", "\t")
			fmt.Println(string(d))
		*/
		embed.Title += " - Hackban"
	} else {
		/*
			d, _ := json.MarshalIndent(cmem, "", "\t")
			fmt.Println(string(d))
			current := time.Now().Unix()
				fmt.Println(current)

				jeff := current - int64((time.Hour * 24).Seconds())
				fmt.Println(strconv.FormatInt(jeff, 10))
		*/
		messagelog := []*DMsg{}

		err = msgDB.View(func(txn *badger.Txn) error {
			it := txn.NewIterator(badger.DefaultIteratorOptions)
			defer it.Close()

			prefix := []byte(fmt.Sprintf("%v:", m.GuildID))
			for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
				item := it.Item()

				body, err := item.ValueCopy(nil)
				if err != nil {
					continue
				}
				dec := gob.NewDecoder(bytes.NewReader(body))
				msg := &DMsg{}
				err = dec.Decode(msg)
				if err != nil {
					continue
				}
				/*
					d, _ := json.MarshalIndent(msg, "", "\t")
					fmt.Println(string(d))
				*/
				if msg.Message.Author.ID == m.User.ID {

					msgid, err := strconv.ParseInt(msg.Message.ID, 10, 0)
					if err != nil {
						continue
					}
					msgts := ((msgid >> 22) + 1420070400000) / 1000

					dayAgo := ts.Unix() - int64((time.Hour * 24).Seconds())

					if msgts > dayAgo {
						messagelog = append(messagelog, msg)
					}
				}

				/*

					unixmsgts := ((msgts >> 22) + 1420070400000) / 1000

					if dayAgo > unixmsgts {
						messagelog = append(messagelog, msg)
					} */
			}
			return nil
		})
		if err != nil {
			fmt.Println(err)
			return
		}

		text := ""

		sort.Sort(ByID(messagelog))

		for _, cmsg := range messagelog {
			if cmsg.Message.Author.ID == m.User.ID {

				ch, err := s.State.Channel(cmsg.Message.ChannelID)
				if err != nil {
					continue
				}

				if len(cmsg.Attachments) > 0 {
					text += fmt.Sprintf("\nUser: %v (%v)\nChannel: %v (%v)\nContent: %v\nMessage had attachment\n", cmsg.Message.Author.String(), cmsg.Message.Author.ID, ch.Name, ch.ID, cmsg.Message.Content)
				} else {
					text += fmt.Sprintf("\nUser: %v (%v)\nChannel: %v (%v)\nContent: %v\n", cmsg.Message.Author.String(), cmsg.Message.Author.ID, ch.Name, ch.ID, cmsg.Message.Content)
				}
			}
		}

		if len(messagelog) > 0 {

			link, err := OWOC.Upload(text)
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

	_, err = s.ChannelMessageSendEmbed(dg.BanLog, &embed)
	if err != nil {
		fmt.Println("BAN LOG ERROR", err)
	}
}

func GuildBanRemoveHandler(s *discordgo.Session, m *discordgo.GuildBanRemove) {
	row := db.QueryRow("SELECT unbanlog FROM discordguilds WHERE guildid=$1;", m.GuildID)
	dg := DiscordGuild{}
	err := row.Scan(&dg.UnbanLog)
	if err != nil {
		fmt.Println(err)
		return
	}
	if dg.UnbanLog == "" {
		return
	}

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

	_, err = s.ChannelMessageSendEmbed(dg.UnbanLog, &embed)
	if err != nil {
		fmt.Println("UNBAN LOG ERROR", err)
	}
}

func MessageUpdateHandler(s *discordgo.Session, m *discordgo.MessageUpdate) {
	row := db.QueryRow("SELECT msgeditlog FROM discordguilds WHERE guildid=$1;", m.GuildID)
	dg := DiscordGuild{}
	err := row.Scan(&dg.MsgEditLog)
	if err != nil {
		fmt.Println(err)
		return
	}
	if dg.MsgEditLog == "" {
		return
	}

	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		return
	}

	oldm, err := GetMessage(fmt.Sprintf("%v:%v:%v", m.GuildID, m.ChannelID, m.ID))
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
	_, err = s.ChannelMessageSendEmbed(dg.MsgEditLog, &embed)
	if err != nil {
		fmt.Println("EDIT LOG ERROR", err)
	}

	oldm.Message.Content = m.Content

	err = LoadMessage(oldm.Message)
	if err != nil {
		fmt.Println("ERROR")
		return
	}
}

func MessageDeleteBulkHandler(s *discordgo.Session, m *discordgo.MessageDeleteBulk) {

	row := db.QueryRow("SELECT msgdeletelog FROM discordguilds WHERE guildid=$1;", m.GuildID)
	dg := DiscordGuild{}
	err := row.Scan(&dg.MsgDeleteLog)
	if err != nil {
		fmt.Println(err)
		return
	}
	if dg.MsgDeleteLog == "" {
		return
	}

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
	deletedmsgs := []*DMsg{}
	for _, msgid := range m.Messages {
		delmsg, err := GetMessage(fmt.Sprintf("%v:%v:%v", m.GuildID, m.ChannelID, msgid))
		if err != nil {
			continue
		}
		deletedmsgs = append(deletedmsgs, delmsg)
	}

	sort.Sort(ByID(deletedmsgs))

	text := fmt.Sprintf("%v - %v\n\n\n", m.ChannelID, ts.Format(time.RFC1123))

	for _, msg := range deletedmsgs {
		if len(msg.Attachments) > 0 {
			text += fmt.Sprintf("\nUser: %v (%v)\nContent: %v\nMessage had attachment\n", msg.Message.Author.String(), msg.Message.Author.ID, msg.Message.Content)
		} else {
			text += fmt.Sprintf("\nUser: %v (%v)\nContent: %v\n", msg.Message.Author.String(), msg.Message.Author.ID, msg.Message.Content)
		}
	}

	res, err := OWOC.Upload(text)

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
	_, err = s.ChannelMessageSendEmbed(config.MsgDelete, &embed)
	if err != nil {
		fmt.Println("BULK DELETE LOG ERROR", err)
	}
}

func MessageDeleteHandler(s *discordgo.Session, m *discordgo.MessageDelete) {
	row := db.QueryRow("SELECT msgdeletelog FROM discordguilds WHERE guildid=$1;", m.GuildID)
	dg := DiscordGuild{}
	err := row.Scan(&dg.MsgDeleteLog)
	if err != nil {
		fmt.Println(err)
		return
	}
	if dg.MsgDeleteLog == "" {
		return
	}

	msg, err := GetMessage(fmt.Sprintf("%v:%v:%v", m.GuildID, m.ChannelID, m.ID))
	if err != nil {
		//fmt.Println(err)
		return
	}
	g, err := s.State.Guild(m.GuildID)
	if err != nil {
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

	_, err = s.ChannelMessageSendEmbed(config.MsgDelete, &embed)
	if err != nil {
		fmt.Println("DELETE LOG ERROR", err)
	}
	if len(msg.Message.Attachments) > 0 {
		send, err := s.ChannelMessageSend(dg.MsgDeleteLog, "Trying to get attachments..")
		if err != nil {
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

		_, err = s.ChannelMessageSendComplex(dg.MsgDeleteLog, data)
		if err != nil {
			s.ChannelMessageEdit(send.ChannelID, send.ID, "Error getting attachments")
			fmt.Println("DELETE LOG ERROR", err)
		} else {
			s.ChannelMessageDelete(send.ChannelID, send.ID)
		}
	}
}

func MessageCreateHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		fmt.Println(err)
		return
	}

	ch, err := s.State.Channel(m.ChannelID)
	if err != nil {
		fmt.Println(err)
		return
	}
	if ch.Type != discordgo.ChannelTypeGuildText {
		return
	}

	fmt.Println(fmt.Sprintf("%v - %v - %v: %v", g.Name, ch.Name, m.Author.String(), m.Content))

	err = LoadMessage(m.Message)
	if err != nil {
		fmt.Println("MESSAGE CREATE ERROR", err)
		return
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
			db.Exec("UPDATE discordguilds SET joinlog=$1 WHERE guildid=$2", channel.ID, g.ID)
			s.ChannelMessageSend(ch.ID, fmt.Sprintf("Set join logs to %v.", channel.Mention()))
		case "leave":
			db.Exec("UPDATE discordguilds SET leavelog=$1 WHERE guildid=$2", channel.ID, g.ID)
			s.ChannelMessageSend(ch.ID, fmt.Sprintf("Set leave logs to %v.", channel.Mention()))
		case "msgdelete":
			db.Exec("UPDATE discordguilds SET msgdeletelog=$1 WHERE guildid=$2", channel.ID, g.ID)
			s.ChannelMessageSend(ch.ID, fmt.Sprintf("Set message delete logs to %v.", channel.Mention()))
		case "msgedit":
			db.Exec("UPDATE discordguilds SET msgeditlog=$1 WHERE guildid=$2", channel.ID, g.ID)
			s.ChannelMessageSend(ch.ID, fmt.Sprintf("Set message edit logs to %v.", channel.Mention()))
		case "ban":
			db.Exec("UPDATE discordguilds SET banlog=$1 WHERE guildid=$2", channel.ID, g.ID)
			s.ChannelMessageSend(ch.ID, fmt.Sprintf("Set ban logs to %v.", channel.Mention()))
		case "unban":
			db.Exec("UPDATE discordguilds SET unbanlog=$1 WHERE guildid=$2", channel.ID, g.ID)
			s.ChannelMessageSend(ch.ID, fmt.Sprintf("Set unban logs to %v.", channel.Mention()))

		default:
			s.ChannelMessageSend(ch.ID, "no")
		}
	} else if args[0] == "fl.help" {

		text := "To set a log channel, do `fl.set <logtype> <channel>`, where channel is optional.\n"
		text += "Logtypes:\n"
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

func ReadyHandler(s *discordgo.Session, r *discordgo.Ready) {
	fmt.Println(fmt.Sprintf("Logged in as %v.", r.User.String()))
}

func DisconnectHandler(s *discordgo.Session, d *discordgo.Disconnect) {
	fmt.Println("DISCONNECTED AT ", time.Now().Format(time.RFC1123))
}
