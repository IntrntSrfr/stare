package loggerdb

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/dgraph-io/badger/options"
	"github.com/intrntsrfr/functional-logger/structs"

	"github.com/bwmarrin/discordgo"
	"github.com/dgraph-io/badger"
)

type LoggerDB struct {
	Db *badger.DB
}

func NewDB() (*LoggerDB, error) {

	os.MkdirAll("./data", 0750)

	opts := badger.DefaultOptions
	opts.Dir = "./data"
	opts.ValueDir = "./data"
	opts.Truncate = true
	opts.ValueLogLoadingMode = options.FileIO
	db, err := badger.Open(opts)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return &LoggerDB{db}, nil
}

func (db *LoggerDB) SetMember(m *discordgo.Member) error {

	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(m)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return db.Db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(fmt.Sprintf("member:%v:%v", m.GuildID, m.User.ID)), buf.Bytes())
	})
}

func (db *LoggerDB) GetMember(key string) (*discordgo.Member, error) {

	body := []byte{}
	err := db.Db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("member:" + key))
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
	msg := &discordgo.Member{}
	err = gob.NewDecoder(bytes.NewReader(body)).Decode(msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (db *LoggerDB) DeleteMember(key string) error {
	return db.Db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte("member:" + key))
	})
}

func (db *LoggerDB) SetMessage(m *discordgo.Message) error {

	msg := &structs.DMsg{
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
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(msg)
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = db.Db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(fmt.Sprintf("message:%v:%v:%v", m.GuildID, m.ChannelID, m.ID)), buf.Bytes())
	})
	return err
}

func (db *LoggerDB) GetMessage(key string) (*structs.DMsg, error) {
	body := []byte{}
	err := db.Db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("message:" + key))
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
	msg := &structs.DMsg{}
	err = gob.NewDecoder(bytes.NewReader(body)).Decode(msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}
