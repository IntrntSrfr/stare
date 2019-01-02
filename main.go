package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
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

var (
	config Config

	msgCache = NewMsgCache()
	memCache = NewMemCache()
	api      *simplepaste.API
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
		go memCache.Put(mem)
	}
}

func GuildMemberUpdateHandler(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {
	if _, ok := memCache.Get(fmt.Sprint("%v:%v", m.GuildID, m.User.ID)); ok {
		go memCache.Put(m.Member)
	}
}

func GuildUnavailableHandler(s *discordgo.Session, g *discordgo.GuildDelete) {
	for _, mem := range g.Members {
		go memCache.Delete(g.ID + mem.User.ID)
	}
}

func ReadyHandler(s *discordgo.Session, r *discordgo.Ready) {
	fmt.Println(fmt.Sprintf("Logged in as %v.", r.User.String()))
}

func DisconnectHandler(s *discordgo.Session, d *discordgo.Disconnect) {
	fmt.Println("DISCONNECTED AT ", time.Now().Format(time.RFC1123))
}
