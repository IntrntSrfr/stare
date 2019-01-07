package main

import "github.com/bwmarrin/discordgo"

type Config struct {
	OwnerID          string `json:"OwnerID"`
	Token            string `json:"Token"`
	OWOApiKey        string `json:"OWOApiKey"`
	ConnectionString string `json:"ConnectionString"`
	MsgEdit          string `json:"MsgEdit"`
	MsgDelete        string `json:"MsgDelete"`
	Ban              string `json:"Ban"`
	Unban            string `json:"Unban"`
	Join             string `json:"Join"`
	Leave            string `json:"Leave"`
}

type DMsg struct {
	Message     *discordgo.Message
	Attachments [][]byte
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
