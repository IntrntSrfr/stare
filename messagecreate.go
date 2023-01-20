package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/bwmarrin/discordgo"
)

func MessageCreateHandler(s *discordgo.Session, m *discordgo.MessageCreate) {

	if m.Author.Bot {
		return
	}

	ch, err := s.State.Channel(m.ChannelID)
	if err != nil {
		fmt.Println("GUILD ERROR", err)
		return
	}

	if ch.Type != discordgo.ChannelTypeGuildText {
		return
	}

	g, err := s.State.Guild(ch.GuildID)
	if err != nil {
		fmt.Println("CHANNEL ERROR", err)
		return
	}

	fmt.Println(fmt.Sprintf("%v - %v - %v: %v", g.Name, ch.Name, m.Author.String(), m.Content))

	dmsg := &DiscMessage{
		Message:    m.Message,
		Attachment: [][]byte{},
	}

	for _, img := range m.Attachments {

		res, _ := http.Get(img.URL)
		if err != nil {
			return
		}

		d, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return
		}

		dmsg.Attachment = append(dmsg.Attachment, d)
	}

	go msgCache.Put(dmsg)

	if m.Content == "fl.len" {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprint(len(msgCache.storage)))
	} else if m.Content == "fl.mlen" {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprint(len(memCache.storage)))
	} else if m.Author.ID == config.OwnerID {
		if m.Content == "fl.clear" {
			go msgCache.Clear()
		}
	}

	go func() {
		cleartime := time.After(24 * time.Hour)

		select {
		case <-cleartime:
			go msgCache.Delete(m.ID)
		}
	}()
}
