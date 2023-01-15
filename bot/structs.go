package bot

import "github.com/bwmarrin/discordgo"

/*
type Config struct {
	Token            string `json:"token"`
	ConnectionString string `json:"connection_string"`
	OwoAPIKey        string `json:"owo_api_key"`
} */

type Color int

const (
	Red    Color = 0xC80000
	Orange       = 0xF08152
	Blue         = 0x61D1ED
	Green        = 0x00C800
	White        = 0xFFFFFF
)

type Guild struct {
	ID           string `json:"id" db:"id"`
	MsgEditLog   string `json:"msg_edit_log" db:"msg_edit_log"`
	MsgDeleteLog string `json:"msg_delete_log" db:"msg_delete_log"`
	BanLog       string `json:"ban_log" db:"ban_log"`
	UnbanLog     string `json:"unban_log" db:"unban_log"`
	JoinLog      string `json:"join_log" db:"join_log"`
	LeaveLog     string `json:"leave_log" db:"leave_log"`
}

const schemaGuild = `
CREATE TABLE IF NOT EXISTS guilds (
	id             TEXT PRIMARY KEY,
	msg_edit_log   text,
	msg_delete_log text,
	ban_log        text,
	unban_log      text,
	join_log       text,
	leave_log      text
);
`

type DiscordMessage struct {
	Message     *discordgo.Message
	Attachments [][]byte
}
type ByID []*DiscordMessage

func (m ByID) Len() int {
	return len(m)
}

func (m ByID) Less(i, j int) bool {
	return m[i].Message.ID < m[j].Message.ID
}

func (m ByID) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}
