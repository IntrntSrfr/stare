package bot

import (
	"bytes"
	"github.com/bwmarrin/discordgo"
	"reflect"
	"testing"
	"time"
)

func TestAddEmbedField(t *testing.T) {
	type args struct {
		e      *discordgo.MessageEmbed
		name   string
		value  string
		inline bool
	}
	tests := []struct {
		name string
		args args
		want *discordgo.MessageEmbed
	}{
		{
			name: "valid test",
			args: args{&discordgo.MessageEmbed{}, "name", "value", false},
			want: &discordgo.MessageEmbed{Fields: []*discordgo.MessageEmbedField{{"name", "value", false}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AddEmbedField(tt.args.e, tt.args.name, tt.args.value, tt.args.inline); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AddEmbedField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddEmbedField2(t *testing.T) {
	want := &discordgo.MessageEmbed{Fields: []*discordgo.MessageEmbedField{{
		Name:   "name",
		Value:  "value",
		Inline: false,
	}}}

	r := &discordgo.MessageSend{Embed: &discordgo.MessageEmbed{}}
	r.Embed = AddEmbedField(r.Embed, "name", "value", false)

	if !reflect.DeepEqual(r.Embed, want) {
		t.Errorf("AddEmbedField() = %v, want %v", r.Embed, want)
	}
}

func TestAddMessageFile(t *testing.T) {
	type args struct {
		m        *discordgo.MessageSend
		filename string
		data     []byte
	}
	tests := []struct {
		name string
		args args
		want *discordgo.MessageSend
	}{
		{
			name: "valid test",
			args: args{&discordgo.MessageSend{}, "content.txt", []byte("hello")},
			want: &discordgo.MessageSend{Files: []*discordgo.File{{
				Name:   "content.txt",
				Reader: bytes.NewBuffer([]byte("hello")),
			}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AddMessageFile(tt.args.m, tt.args.filename, tt.args.data); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AddMessageFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddMessageFileString(t *testing.T) {
	type args struct {
		m        *discordgo.MessageSend
		filename string
		data     string
	}
	tests := []struct {
		name string
		args args
		want *discordgo.MessageSend
	}{
		{
			name: "valid test",
			args: args{&discordgo.MessageSend{}, "content.txt", "hello"},
			want: &discordgo.MessageSend{Files: []*discordgo.File{{
				Name:   "content.txt",
				Reader: bytes.NewBuffer([]byte("hello")),
			}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AddMessageFileString(tt.args.m, tt.args.filename, tt.args.data); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AddMessageFileString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewLogEmbed(t *testing.T) {
	type args struct {
		t LogType
		g *discordgo.Guild
	}
	tests := []struct {
		name string
		args args
		want *discordgo.MessageEmbed
	}{
		{
			name: "valid test",
			args: args{GuildJoinType, &discordgo.Guild{Name: "jeff", ID: "1234", Icon: "4321"}},
			want: &discordgo.MessageEmbed{Title: "User joined", Footer: &discordgo.MessageEmbedFooter{Text: "jeff", IconURL: discordgo.EndpointGuildIcon("1234", "4321")}},
		},
		{
			name: "valid test, no guild",
			args: args{GuildJoinType, nil},
			want: &discordgo.MessageEmbed{Title: "User joined", Footer: nil},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewLogEmbed(tt.args.t, tt.args.g)
			if got.Title != tt.want.Title {
				t.Errorf("NewLogEmbed() = %v, want %v", got.Title, tt.want.Title)
			}
			if !reflect.DeepEqual(got.Footer, tt.want.Footer) {
				t.Errorf("NewLogEmbed() = %v, want %v", got.Title, tt.want.Title)
			}
		})
	}
}

func TestParseSnowflake(t *testing.T) {
	type args struct {
		id string
	}
	tests := []struct {
		name    string
		args    args
		want    time.Time
		wantErr bool
	}{
		{
			name:    "valid test",
			args:    args{"163454407999094786"},
			want:    time.Unix(1459040967, 0),
			wantErr: false,
		},
		{
			name:    "invalid test",
			args:    args{"asdf"},
			want:    time.Now(),
			wantErr: true,
		},
	}
	abs := func(a time.Duration) time.Duration {
		if a < 0 {
			return -a
		}
		return a
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSnowflake(tt.args.id)
			if err == nil && tt.wantErr {
				t.Errorf("ParseSnowflake() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !tt.wantErr {
				t.Errorf("ParseSnowflake() got = %v, want %v", got, tt.want)
			}
			if err == nil && got != time.Unix(1459040967, 0) {
				t.Errorf("ParseSnowflake() got = %v, want %v", got, tt.want)
			}
			if err != nil && abs(got.Sub(time.Now())) > 5*time.Second {
				t.Errorf("ParseSnowflake() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetEmbedThumbnail(t *testing.T) {
	type args struct {
		e   *discordgo.MessageEmbed
		url string
	}
	tests := []struct {
		name string
		args args
		want *discordgo.MessageEmbed
	}{
		{
			name: "valid test",
			args: args{&discordgo.MessageEmbed{}, "github.com"},
			want: &discordgo.MessageEmbed{Thumbnail: &discordgo.MessageEmbedThumbnail{URL: "github.com"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SetEmbedThumbnail(tt.args.e, tt.args.url); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SetEmbedThumbnail() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTrimChannelString(t *testing.T) {

	tests := []struct {
		name string
		args string
		want string
	}{
		{
			name: "valid test",
			args: "<#1234>",
			want: "1234",
		},
		{
			name: "valid test 2",
			args: "1234",
			want: "1234",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TrimChannelString(tt.args); got != tt.want {
				t.Errorf("TrimChannelString() = %v, want %v", got, tt.want)
			}
		})
	}
}
