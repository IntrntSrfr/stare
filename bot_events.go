package stare

import (
	"github.com/intrntsrfr/meido/pkg/mio"
	"go.uber.org/zap"
)

func logApplicationCommandRan(m *Bot) func(cmd *mio.ApplicationCommandRan) {
	return func(cmd *mio.ApplicationCommandRan) {
		m.logger.Info("Slash",
			zap.String("name", cmd.Interaction.Name()),
			zap.String("id", cmd.Interaction.ID()),
			zap.String("channelID", cmd.Interaction.ChannelID()),
			zap.String("userID", cmd.Interaction.AuthorID()),
		)
	}
}

func logApplicationCommandPanicked(m *Bot) func(cmd *mio.ApplicationCommandPanicked) {
	return func(cmd *mio.ApplicationCommandPanicked) {
		m.logger.Error("Slash panic",
			zap.Any("slash", cmd.ApplicationCommand),
			zap.Any("interaction", cmd.Interaction),
			zap.Any("reason", cmd.Reason),
		)
	}
}
