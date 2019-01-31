package loggerdb

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/dgraph-io/badger/options"

	"github.com/bwmarrin/discordgo"
	"github.com/dgraph-io/badger"
)

type DB struct {
	db *badger.DB
}

func NewDB() (*DB, error) {

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
	return &DB{db}, nil
}

func (db *DB) Close() {
	db.db.Close()
}

func (db *DB) SetMember(m *discordgo.Member) error {

	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(m)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return db.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(fmt.Sprintf("member:%v:%v", m.GuildID, m.User.ID)), buf.Bytes())
	})
}

func (db *DB) GetMember(key string) (*discordgo.Member, error) {

	body := []byte{}
	err := db.db.View(func(txn *badger.Txn) error {
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

func (db *DB) DeleteMember(key string) error {
	return db.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte("member:" + key))
	})
}

func (db *DB) SetMessage(m *discordgo.Message) error {

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
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(msg)
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = db.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(fmt.Sprintf("message:%v:%v:%v", m.GuildID, m.ChannelID, m.ID)), buf.Bytes())
	})
	return err
}

func (db *DB) GetMessage(key string) (*DMsg, error) {
	body := []byte{}
	err := db.db.View(func(txn *badger.Txn) error {
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
	msg := &DMsg{}
	err = gob.NewDecoder(bytes.NewReader(body)).Decode(msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (db *DB) GetMessageLog(messagelog []*DMsg, m *discordgo.GuildBanAdd) error {
	return db.db.View(func(txn *badger.Txn) error {
		ts := time.Now()
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		prefix := []byte(fmt.Sprintf("%v:", m.GuildID))
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()

			body, err := item.ValueCopy(nil)
			if err != nil {
				continue
			}
			msg := &DMsg{}
			err = gob.NewDecoder(bytes.NewReader(body)).Decode(msg)
			if err != nil {
				continue
			}

			if msg.Message.Author.ID == m.User.ID {

				msgid, err := strconv.ParseInt(msg.Message.ID, 10, 0)
				if err != nil {
					continue
				}
				msgts := ((msgid >> 22) + 1420070400000) / 1000

				dayAgo := ts.Unix() - int64((time.Hour * 24).Seconds())

				if msgts > dayAgo {
					messagelog = append(messagelog, msg)
				}
			}

			/*

				unixmsgts := ((msgts >> 22) + 1420070400000) / 1000

				if dayAgo > unixmsgts {
					messagelog = append(messagelog, msg)
				} */
		}
		return nil
	})
}
