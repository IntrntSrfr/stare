package discord

import (
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
	"net/http"
)

type Discord struct {
	token    string
	Sess     *discordgo.Session
	sessions []*discordgo.Session
	log      *zap.Logger

	Events chan interface{}
}

// NewDiscord takes in a token and creates a Discord object.
func NewDiscord(token string, log *zap.Logger) (*Discord, error) {
	d := &Discord{
		token:  token,
		log:    log,
		Events: make(chan interface{}, 256),
	}

	shardCount, err := recommendedShards(d.token)
	if err != nil {
		return nil, err
	}

	for i := 0; i < shardCount; i++ {
		s, err := discordgo.New("Bot " + d.token)
		if err != nil {
			return nil, err
		}

		s.State.TrackVoice = false
		s.State.TrackPresences = false
		s.ShardCount = shardCount
		s.ShardID = i
		s.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsAllWithoutPrivileged | discordgo.IntentsGuildMembers | discordgo.IntentMessageContent)
		/*
			s.AddHandler(onGuildCreate(d.Events))
			s.AddHandler(onGuildMemberUpdate(d.Events))
			s.AddHandler(onGuildMemberAdd(d.Events))
			s.AddHandler(onGuildMemberRemove(d.Events))
			s.AddHandler(onGuildMembersChunk(d.Events))
			s.AddHandler(onGuildBanAdd(d.Events))
			s.AddHandler(onGuildBanRemove(d.Events))
			s.AddHandler(onMessageCreate(d.Events))
			s.AddHandler(onMessageUpdate(d.Events))
			s.AddHandler(onMessageDelete(d.Events))
			s.AddHandler(onMessageDeleteBulk(d.Events))
			s.AddHandler(onReady(d.Events))
			s.AddHandler(onDisconnect(d.Events))
		*/
		s.AddHandler(onEvent(d.Events))

		d.sessions = append(d.sessions, s)
		fmt.Println("created session:", i)
	}
	d.Sess = d.sessions[0]

	return d, nil
}

func onEvent(e chan interface{}) func(s *discordgo.Session, i interface{}) {
	return func(s *discordgo.Session, i interface{}) {
		e <- i
	}
}

func (d *Discord) AddHandler(h interface{}) {
	for _, s := range d.sessions {
		s.AddHandler(h)
	}
}

// Open opens the Discord sessions.
func (d *Discord) Open() error {
	for _, sess := range d.sessions {
		if err := sess.Open(); err != nil {
			return err
		}
	}
	return nil
}

// Close closes the Discord sessions
func (d *Discord) Close() {
	for _, sess := range d.sessions {
		if err := sess.Close(); err != nil {
			d.log.Error("failed to close discord session", zap.Int("shard", sess.ShardID), zap.Error(err))
		}
	}
}

// recommendedShards asks discord for the recommended shardcount for the bot given the token.
// returns -1 if the request does not go well.
func recommendedShards(token string) (int, error) {
	req, _ := http.NewRequest("GET", "https://discord.com/api/v8/gateway/bot", nil)
	req.Header.Add("Authorization", "Bot "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return -1, err
	}
	defer res.Body.Close()

	resp := &discordgo.GatewayBotResponse{}
	err = json.NewDecoder(res.Body).Decode(&resp)
	if err != nil {
		return -1, err
	}

	return resp.Shards, nil
}
