package bot

import "github.com/bwmarrin/discordgo"

type Color int

const (
	Red    Color = 0xC80000
	Orange       = 0xF08152
	Blue         = 0x61D1ED
	Green        = 0x00C800
	White        = 0xFFFFFF
)

type Context struct {
	Sess  *discordgo.Session
	Guild *Guild
	Event interface{}
}

type Guild struct {
	ID           string `json:"id" db:"id"`
	MsgEditLog   string `json:"msg_edit_log" db:"msg_edit_log"`
	MsgDeleteLog string `json:"msg_delete_log" db:"msg_delete_log"`
	BanLog       string `json:"ban_log" db:"ban_log"`
	UnbanLog     string `json:"unban_log" db:"unban_log"`
	JoinLog      string `json:"join_log" db:"join_log"`
	LeaveLog     string `json:"leave_log" db:"leave_log"`
}
