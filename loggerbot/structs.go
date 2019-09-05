package loggerbot

type Config struct {
	Token            string `json:"token"`
	ConnectionString string `json:"connection_string"`
	OwoAPIKey        string `json:"owo_api_key"`
}

const (
	dColorRed    = 13107200
	dColorOrange = 15761746
	dColorLBlue  = 6410733
	dColorGreen  = 51200
	dColorWhite  = 16777215
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
