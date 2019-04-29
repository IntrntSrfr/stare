package loggerbot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

func (b *Bot) guildCreateHandler(s *discordgo.Session, g *discordgo.GuildCreate) {

	b.logger.Info("EVENT: GUILD CREATE")

	var count int

	row := b.db.QueryRow("SELECT COUNT(*) FROM discordguilds WHERE guildid = $1;", g.ID)

	err := row.Scan(&count)
	if err != nil {
		b.logger.Error("error", zap.Error(err))
		return
	}
	if count == 0 {
		_, err := b.db.Exec("INSERT INTO discordguilds(guildid, msgeditlog, msgdeletelog, banlog, unbanlog, joinlog, leavelog) VALUES($1, $2, $3, $4, $5, $6, $7);", g.ID, "", "", "", "", "", "")
		if err != nil {
			fmt.Println(err)
			b.logger.Error("error", zap.Error(err))
			return
		}
	}

	if len(g.Members) != g.MemberCount {
		s.RequestGuildMembers(g.ID, "", 0)
	} else {
		go func() {
			for _, mem := range g.Members {

				err := b.loggerDB.SetMember(mem, 1)
				if err != nil {
					b.logger.Error("error", zap.Error(err))
					continue
				}
			}
		}()
	}

	owner := ""
	own, err := s.State.Member(g.ID, g.OwnerID)
	if err != nil {
		owner = g.OwnerID
	} else {
		owner = own.User.String()
	}

	b.logger.Info(fmt.Sprintf("LOADED %v - %v", g.Name, owner))
	fmt.Println(fmt.Sprintf("LOADED %v - %v", g.Name, owner))
}
