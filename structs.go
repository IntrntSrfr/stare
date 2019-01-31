package main

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
