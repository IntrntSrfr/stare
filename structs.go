package main

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
