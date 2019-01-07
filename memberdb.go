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

	err = msgDB.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte(fmt.Sprintf("%v:%v", m.GuildID, m.User.ID)), buf.Bytes())
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

func GetMember(key string) (*discordgo.Member, error) {

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
	msg := &discordgo.Member{}
	dec.Decode(msg)
	return msg, nil
}

func DeleteMember(key string) error {
	err := memDB.Update(func(txn *badger.Txn) error {
		err := txn.Delete([]byte(key))
		if err != nil {
			fmt.Println(err)
			return err
		} else {
			fmt.Println("deleted", key)
		}
		return nil
	})
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}
