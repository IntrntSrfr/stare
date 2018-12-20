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
	"unicode/utf8"

	"github.com/ninedraft/simplepaste"

	"github.com/bwmarrin/discordgo"
)

type Config struct {
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
	go s.AddHandler(GuildAvailableHandler)
	go s.AddHandler(GuildMemberUpdateHandler)
	go s.AddHandler(MemberJoinedHandler)
	go s.AddHandler(MemberLeaveHandler)
	go s.AddHandler(MemberBannedHandler)
	go s.AddHandler(MemberUnbannedHandler)
	go s.AddHandler(MessageUpdateHandler)
	go s.AddHandler(MessageDeleteHandler)
	go s.AddHandler(MessageDeleteBulkHandler)
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

func MemberJoinedHandler(s *discordgo.Session, m *discordgo.GuildMemberAdd) {

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
	}

	s.ChannelMessageSendEmbed(config.Join, &embed)

}
func MemberLeaveHandler(s *discordgo.Session, m *discordgo.GuildMemberRemove) {

	gamer := []string{}

	if mem, ok := memberCache[m.GuildID+m.User.ID]; ok {

		for _, r := range mem.Roles {
			gamer = append(gamer, fmt.Sprintf("<@&%v>", r))
		}

		fmt.Println(gamer)

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

		_, err := s.ChannelMessageSendEmbed(config.Leave, &embed)
		if err != nil {
			fmt.Println("ERROR", err)
		}

		delete(memberCache, m.GuildID+m.User.ID)
	}
}

func MemberBannedHandler(s *discordgo.Session, m *discordgo.GuildBanAdd) {

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
	}

	text := ""

	for _, mc := range messageCache {
		if mc.message.Author.ID == m.User.ID {
			if len(mc.attachment) > 0 {
				text += fmt.Sprintf("\nUser: %v (%v)\nContent: %v\nMessage had attachment\n", mc.message.Author.String(), mc.message.Author.ID, mc.message.Content)
			} else {
				text += fmt.Sprintf("\nUser: %v (%v)\nContent: %v\n", mc.message.Author.String(), mc.message.Author.ID, mc.message.Content)
			}
		}
	}

	ts := time.Now()

	paste := simplepaste.NewPaste(fmt.Sprintf("24h ban log for %v (%v) - %v", m.User.String(), m.User.ID, ts.Format(time.RFC1123)), text)

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

	s.ChannelMessageSendEmbed(config.Ban, &embed)

}
func MemberUnbannedHandler(s *discordgo.Session, m *discordgo.GuildBanRemove) {

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
	}

	s.ChannelMessageSendEmbed(config.Unban, &embed)
}

func MessageUpdateHandler(s *discordgo.Session, m *discordgo.MessageUpdate) {

	if oldm, ok := messageCache[m.ID]; ok {
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
		}
		s.ChannelMessageSendEmbed(config.MsgEdit, &embed)
	}
}

func MessageDeleteHandler(s *discordgo.Session, m *discordgo.MessageDelete) {

	if msg, ok := messageCache[m.ID]; ok {
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

		_, err := s.ChannelMessageSendEmbed(config.MsgDelete, &embed)
		if err != nil {
			fmt.Println(err)
		}
		if len(msgo.Attachments) > 0 {
			//ext := strings.Split()
			s.ChannelFileSendWithMessage(config.MsgDelete, fmt.Sprintf("File attached to message ID: %v", m.ID), msgo.Attachments[0].Filename, bytes.NewReader(msg.attachment))
		}
	}
}

func MessageDeleteBulkHandler(s *discordgo.Session, m *discordgo.MessageDeleteBulk) {

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
		fmt.Println("ERROR", err)
	}
}

func MessageCreateHandler(s *discordgo.Session, m *discordgo.MessageCreate) {

	if m.Author.Bot {
		return
	}

	ch, err := s.Channel(m.ChannelID)
	if err != nil {
		fmt.Println("GUILD ERROR", err)
		return
	}

	if ch.Type != discordgo.ChannelTypeGuildText {
		return
	}

	g, err := s.Guild(ch.GuildID)
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
	}

	args := strings.Split(m.Content, " ")

	if len(args) > 0 {
		perms, err := s.State.UserChannelPermissions(m.Author.ID, ch.ID)
		if err != nil {
			return
		}

		if perms&discordgo.PermissionManageNicknames == 0 && perms&discordgo.PermissionAdministrator == 0 {
			//fmt.Println(perms, cmd.RequiredPerms, permMap[cmd.RequiredPerms], perms&cmd.RequiredPerms)
			return
		}

		if args[0] == "m?cnb" {
			go coolNameBro(s, args, ch, g)
		}
	}

	cleartime := time.After(24 * time.Hour)

	select {
	case <-cleartime:
		delete(messageCache, m.ID)
	}

}
func ReadyHandler(s *discordgo.Session, r *discordgo.Ready) {

	fmt.Println(fmt.Sprintf("Logged in as %v.", r.User.String()))

}
func DisconnectHandler(s *discordgo.Session, d *discordgo.Disconnect) {

}

func coolNameBro(s *discordgo.Session, args []string, ch *discordgo.Channel, g *discordgo.Guild) {

	if len(args) < 2 {
		s.ChannelMessageSend(ch.ID, "Please choose a proper name.")
		return
	}

	newName := strings.Join(args[1:], " ")

	memberList := []string{}

	f, err := os.Open("./ranges.json")
	if err != nil {
		s.ChannelMessageSend(ch.ID, "Cannot find ranges.json")
		return
	}
	defer f.Close()
	ich := charRanges{}

	json.NewDecoder(f).Decode(&ich)

	for _, val := range g.Members {
		if badName(val, &ich) {
			memberList = append(memberList, val.User.ID)
		}
	}

	if len(memberList) < 1 {
		s.ChannelMessageSend(ch.ID, "There is no one rename.")
		return
	} else {
		s.ChannelMessageSend(ch.ID, fmt.Sprintf("Starting rename of %v user(s).", len(memberList)))
	}

	var successfulRenames, failedRenames int

	for _, val := range memberList {
		err := s.GuildMemberNickname(g.ID, val, newName)
		if err != nil {
			failedRenames++
		} else {
			successfulRenames++
		}
	}

	s.ChannelMessageSend(ch.ID, fmt.Sprintf("Rename finished. Successful: %v. Failed: %v.", successfulRenames, failedRenames))
}

func badName(u *discordgo.Member, ich *charRanges) bool {
	isIllegal := false

	if u.Nick != "" {
		r, _ := utf8.DecodeRuneInString(u.Nick)
		for _, rng := range ich.Ranges {
			isIllegal = rng.Start <= int(r) && int(r) <= rng.Stop
			if isIllegal {
				break
			}
		}
	} else {
		r, _ := utf8.DecodeRuneInString(u.User.Username)
		for _, rng := range ich.Ranges {
			isIllegal = rng.Start <= int(r) && int(r) <= rng.Stop
			if isIllegal {
				break
			}
		}
	}
	return isIllegal
}

type charRanges struct {
	Ranges []struct {
		Start int `json:"start"`
		Stop  int `json:"stop"`
	} `json:"ranges"`
}
