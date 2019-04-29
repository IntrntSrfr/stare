package loggerbot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

func (b *Bot) guildMembersChunkHandler(s *discordgo.Session, g *discordgo.GuildMembersChunk) {

	go func() {
		for _, mem := range g.Members {

			err := b.loggerDB.SetMember(mem, 1)
			if err != nil {
				b.logger.Error("error", zap.Error(err))
				continue
			}

		}

		sg, err := s.State.Guild(g.GuildID)
		if err != nil {
			b.logger.Error("error", zap.Error(err))
			return
		}

		b.logger.Info(fmt.Sprintf("UPDATED %v MEMBERS", sg.Name))
		fmt.Println(fmt.Sprintf("UPDATED %v MEMBERS", sg.Name))
	}()
}
