package discord

import "github.com/bwmarrin/discordgo"

func (d *Discord) Guild(gid string) (*discordgo.Guild, error) {
	for _, s := range d.sessions {
		if g, err := s.State.Guild(gid); err == nil {
			return g, nil
		}
	}
	return nil, discordgo.ErrStateNotFound
}

func (d *Discord) Member(gid, uid string) (*discordgo.Member, error) {
	for _, s := range d.sessions {
		if m, err := s.State.Member(gid, uid); err == nil {
			return m, nil
		}
	}
	return nil, discordgo.ErrStateNotFound
}

func (d *Discord) Channel(cid string) (*discordgo.Channel, error) {
	for _, s := range d.sessions {
		if ch, err := s.State.Channel(cid); err == nil {
			return ch, nil
		}
	}
	return nil, discordgo.ErrStateNotFound
}

func (d *Discord) UserChannelPermissions(gid, cid string) (int64, error) {
	for _, s := range d.sessions {
		if p, err := s.State.UserChannelPermissions(gid, cid); err == nil {
			return p, nil
		}
	}
	return -1, discordgo.ErrStateNotFound
}
