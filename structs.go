package stare

import (
	"io"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

type DiscordMessage struct {
	Message     *discordgo.Message
	Attachments []*Attachment
}

func NewDiscordMessage(msg *discordgo.Message, maxSize int) *DiscordMessage {
	m := &DiscordMessage{
		Message:     msg,
		Attachments: []*Attachment{},
	}

	for _, a := range msg.Attachments {
		if a.Size > maxSize {
			continue
		}

		data, err := GetAttachment(a.URL)
		if err != nil {
			continue
		}

		m.Attachments = append(m.Attachments, &Attachment{
			Filename: a.Filename,
			Size:     a.Size,
			Data:     data,
		})
	}
	return m
}

func GetAttachment(url string) ([]byte, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	return io.ReadAll(res.Body)
}

type Attachment struct {
	Filename string
	Size     int
	Data     []byte
}
