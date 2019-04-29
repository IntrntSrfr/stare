package loggerbot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

func (b *Bot) guildMemberUpdateHandler(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {

	err := b.loggerDB.SetMember(m.Member, 0)
	if err != nil {
		fmt.Println(err)
		b.logger.Info("error", zap.Error(err))
		return
	}
}
