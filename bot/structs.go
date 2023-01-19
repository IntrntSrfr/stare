package bot

import (
	"github.com/bwmarrin/discordgo"
	"github.com/intrntsrfr/functional-logger/database"
)

type Color int

const (
	Red    Color = 0xC80000
	Orange       = 0xF08152
	Blue         = 0x61D1ED
	Green        = 0x00C800
	White        = 0xFFFFFF
)

type Context struct {
	b  *Bot
	s  *discordgo.Session
	gc *database.Guild
}
