package structs

import (
	"github.com/bwmarrin/discordgo"
)

type Config struct {
	Token     string `json:"Token"`
	OWOApiKey string `json:"OWOApiKey"`
	MsgEdit   string `json:"MsgEdit"`
	MsgDelete string `json:"MsgDelete"`
	Ban       string `json:"Ban"`
	Unban     string `json:"Unban"`
	Join      string `json:"Join"`
	Leave     string `json:"Leave"`
}

type DMsg struct {
	Message     *discordgo.Message
	Attachments [][]byte
}
type ByID []*DMsg

func (m ByID) Len() int {
	return len(m)
}

func (m ByID) Less(i, j int) bool {
	return m[i].Message.ID < m[j].Message.ID
}

func (m ByID) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

type DiscordGuild struct {
	Uid          int
	Guildid      string
	MsgEditLog   string
	MsgDeleteLog string
	BanLog       string
	UnbanLog     string
	JoinLog      string
	LeaveLog     string
}

type OWOResult struct {
	Success bool `json:"success"`
	Files   []struct {
		Hash string `json:"hash"`
		Name string `json:"name"`
		URL  string `json:"url"`
		Size int    `json:"size"`
	} `json:"files"`
}