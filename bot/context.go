package bot

import (
	"github.com/bwmarrin/discordgo"
	"github.com/intrntsrfr/functional-logger/database"
	"github.com/intrntsrfr/functional-logger/discord"
)

type Context struct {
	b  *Bot
	s  *discordgo.Session
	d  *discord.Discord
	gc *database.Guild
}
