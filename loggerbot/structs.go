package loggerbot

type Config struct {
	Token     string `json:"token"`
	OwoAPIKey string `json:"owo_api_key"`
	MsgEdit   string `json:"msg_edit"`
	MsgDelete string `json:"msg_delete"`
	Ban       string `json:"ban"`
	Unban     string `json:"unban"`
	Join      string `json:"join"`
	Leave     string `json:"leave"`
}

const (
	dColorRed    = 13107200
	dColorOrange = 15761746
	dColorLBlue  = 6410733
	dColorGreen  = 51200
	dColorWhite  = 16777215
)
