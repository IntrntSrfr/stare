package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/dgraph-io/badger"
)

func NewMemberDB() (*badger.DB, error) {

	os.MkdirAll("./tmp/mem", 0750)

	opts := badger.DefaultOptions
	opts.Dir = "./tmp/mem"
	opts.ValueDir = "./tmp/mem"
	opts.Truncate = true
	db, err := badger.Open(opts)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return db, nil
}

func LoadMember(m *discordgo.Member) error {

	var buf bytes.Buffer

	enc := gob.NewEncoder(&buf)

	err = enc.Encode(m)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return memDB.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(fmt.Sprintf("%v:%v", m.GuildID, m.User.ID)), buf.Bytes())
	})

}

func GetMember(key string) (*discordgo.Member, error) {

	body := []byte{}
	err := memDB.View(func(txn *badger.Txn) error {
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
	msg := &discordgo.Member{}
	dec.Decode(msg)
	return msg, nil
}

func DeleteMember(key string) error {
	return memDB.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}
