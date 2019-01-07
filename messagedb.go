package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/bwmarrin/discordgo"

	"github.com/dgraph-io/badger"
)

func NewMessageDB() (*badger.DB, error) {

	os.MkdirAll("./tmp/msg", 0750)

	opts := badger.DefaultOptions
	opts.Dir = "./tmp/msg"
	opts.ValueDir = "./tmp/msg"
	db, err := badger.Open(opts)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return db, nil
}

func LoadMessage(m *discordgo.Message) error {
	var buf bytes.Buffer

	enc := gob.NewEncoder(&buf)
	msg := &DMsg{
		Message:     m,
		Attachments: [][]byte{},
	}
	for _, val := range m.Attachments {
		res, err := http.Get(val.URL)
		if err != nil {
			continue
		}
		defer res.Body.Close()

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			continue
		}

		msg.Attachments = append(msg.Attachments, body)
	}
	err = enc.Encode(msg)
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = msgDB.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte(fmt.Sprintf("%v:%v:%v", m.GuildID, m.ChannelID, m.ID)), buf.Bytes())
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

func GetMessage(key string) (*DMsg, error) {
	body := []byte{}
	err := msgDB.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		body, err = item.ValueCopy(nil)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	dec := gob.NewDecoder(bytes.NewReader(body))
	msg := &DMsg{}
	dec.Decode(msg)
	return msg, nil
}

type ByID []*DMsg

func (m ByID) Len() int {
	return len(m)
}

func (m ByID) Less(i, j int) bool {
	return m[i].Message.ID < m[j].Message.ID
}

func (m ByID) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}
