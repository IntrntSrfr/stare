package loggerdb

import "github.com/bwmarrin/discordgo"

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
