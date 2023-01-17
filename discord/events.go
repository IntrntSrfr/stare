package discord

import "github.com/bwmarrin/discordgo"

func onReady(e chan interface{}) func(s *discordgo.Session, r *discordgo.Ready) {
	return func(s *discordgo.Session, r *discordgo.Ready) {
		e <- r
	}
}

func onDisconnect(e chan interface{}) func(s *discordgo.Session, d *discordgo.Disconnect) {
	return func(s *discordgo.Session, d *discordgo.Disconnect) {
		e <- d
	}
}

func onMessageDeleteBulk(e chan interface{}) func(s *discordgo.Session, m *discordgo.MessageDeleteBulk) {
	return func(s *discordgo.Session, m *discordgo.MessageDeleteBulk) {
		e <- m
	}
}

func onMessageDelete(e chan interface{}) func(s *discordgo.Session, m *discordgo.MessageDelete) {
	return func(s *discordgo.Session, m *discordgo.MessageDelete) {
		e <- m
	}
}

func onMessageUpdate(e chan interface{}) func(s *discordgo.Session, m *discordgo.MessageUpdate) {
	return func(s *discordgo.Session, m *discordgo.MessageUpdate) {
		e <- m
	}
}

func onMessageCreate(e chan interface{}) func(s *discordgo.Session, m *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		e <- m
	}
}

func onGuildBanRemove(e chan interface{}) func(s *discordgo.Session, m *discordgo.GuildBanRemove) {
	return func(s *discordgo.Session, m *discordgo.GuildBanRemove) {
		e <- m
	}
}

func onGuildBanAdd(e chan interface{}) func(s *discordgo.Session, m *discordgo.GuildBanAdd) {
	return func(s *discordgo.Session, m *discordgo.GuildBanAdd) {
		e <- m
	}
}

func onGuildMembersChunk(e chan interface{}) func(s *discordgo.Session, g *discordgo.GuildMembersChunk) {
	return func(s *discordgo.Session, g *discordgo.GuildMembersChunk) {
		e <- g
	}
}

func onGuildMemberRemove(e chan interface{}) func(s *discordgo.Session, m *discordgo.GuildMemberRemove) {
	return func(s *discordgo.Session, m *discordgo.GuildMemberRemove) {
		e <- m
	}
}

func onGuildMemberAdd(e chan interface{}) func(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	return func(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
		e <- m
	}
}

func onGuildMemberUpdate(e chan interface{}) func(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {
	return func(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {
		e <- m
	}

}

func onGuildCreate(e chan interface{}) func(s *discordgo.Session, g *discordgo.GuildCreate) {
	return func(s *discordgo.Session, g *discordgo.GuildCreate) {
		e <- g
	}
}
