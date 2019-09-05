package loggerdb

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/dgraph-io/badger/options"

	"github.com/bwmarrin/discordgo"
	"github.com/dgraph-io/badger"
)

type DB struct {
	db            *badger.DB
	log           *zap.Logger
	TotalMessages int64
	TotalMembers  int64
}

func NewDB(zlog *zap.Logger) (*DB, error) {

	os.MkdirAll("./data", 0750)

	opts := badger.DefaultOptions("./data")
	opts.Truncate = true
	opts.ValueLogLoadingMode = options.FileIO
	opts.NumVersionsToKeep = 1
	db, err := badger.Open(opts)
	if err != nil {
		fmt.Println(err)
		zlog.Info("error", zap.Error(err))
		return nil, err
	}
	zlog.Info("created loggerDB")
	return &DB{db, zlog, 0, 0}, nil
}

func (db *DB) Close() {
	db.setTotal()
	db.db.Close()
}

func (db *DB) setTotal() error {
	return db.db.Update(func(txn *badger.Txn) error {
		var b [8]byte
		binary.BigEndian.PutUint64(b[:], uint64(db.TotalMessages))
		return txn.Set([]byte("TotalMessages"), b[:])
	})
}

func (db *DB) LoadTotal() error {

	var msgBody []byte
	err := db.db.View(func(txn *badger.Txn) error {

		msgs, err := txn.Get([]byte("TotalMessages"))
		if err != nil {
			return err
		}
		msgBody, err = msgs.ValueCopy(nil)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		db.log.Info("error", zap.Error(err))
		return err
	}

	db.TotalMessages = int64(binary.BigEndian.Uint64(msgBody))
	return nil
}

func (db *DB) SetMember(m *discordgo.Member, delta int64) error {

	atomic.AddInt64(&db.TotalMembers, delta)

	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(m)
	if err != nil {
		db.log.Info("error", zap.Error(err))
		return err
	}

	return db.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(fmt.Sprintf("member:%v:%v", m.GuildID, m.User.ID)), buf.Bytes())
	})
}

func (db *DB) GetMember(key string) (*discordgo.Member, error) {

	var body []byte
	err := db.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("member:" + key))
		if err != nil {
			if err != badger.ErrKeyNotFound {
				db.log.Info("error", zap.Error(err))
			}
			//db.log.Info(fmt.Sprintf("Key not found: %v", "member:"+key))
			return err
		}
		body, err = item.ValueCopy(nil)
		if err != nil {
			db.log.Info("error", zap.Error(err))
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
		db.log.Info("error", zap.Error(err))
		return nil, err
	}
	return msg, nil
}

func (db *DB) DeleteMember(key string) error {

	atomic.AddInt64(&db.TotalMembers, -1)

	return db.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte("member:" + key))
	})
}

func (db *DB) SetMessage(m *discordgo.Message) error {

	atomic.AddInt64(&db.TotalMessages, 1)

	msg := &DMsg{
		Message:     m,
		Attachments: [][]byte{},
	}
	for _, val := range m.Attachments {
		res, err := http.Get(val.URL)
		if err != nil {
			db.log.Info("error", zap.Error(err))
			continue
		}
		defer res.Body.Close()

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			db.log.Info("error", zap.Error(err))
			continue
		}

		msg.Attachments = append(msg.Attachments, body)
	}
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(msg)
	if err != nil {
		db.log.Info("error", zap.Error(err))
		return err
	}

	err = db.db.Update(func(txn *badger.Txn) error {
		e := badger.Entry{
			Key:       []byte(fmt.Sprintf("message:%v:%v:%v", m.GuildID, m.ChannelID, m.ID)),
			Value:     buf.Bytes(),
			ExpiresAt: uint64(time.Now().Add(time.Hour * 24).Unix()),
		}
		return txn.SetEntry(&e)
	})
	return err
}

func (db *DB) GetMessage(key string) (*DMsg, error) {
	var body []byte
	err := db.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("message:" + key))
		if err != nil {
			if err != badger.ErrKeyNotFound {
				db.log.Info("error", zap.Error(err))
			}
			//db.log.Info(fmt.Sprintf("Key not found: %v", "message:"+key))
			return err
		}
		body, err = item.ValueCopy(nil)
		if err != nil {
			db.log.Info("error", zap.Error(err))
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
		db.log.Info("error", zap.Error(err))
		return nil, err
	}
	return msg, nil
}

func (db *DB) GetMessageLog(m *discordgo.GuildBanAdd) ([]*DMsg, error) {
	messagelog := []*DMsg{}
	err := db.db.View(func(txn *badger.Txn) error {
		ts := time.Now()
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		prefix := []byte(fmt.Sprintf("message:%v:", m.GuildID))
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()

			body, err := item.ValueCopy(nil)
			if err != nil {
				db.log.Info("error", zap.Error(err))
				continue
			}
			msg := &DMsg{}
			err = gob.NewDecoder(bytes.NewReader(body)).Decode(msg)
			if err != nil {
				db.log.Info("error", zap.Error(err))
				continue
			}

			if msg.Message.Author.ID == m.User.ID {

				msgid, err := strconv.ParseInt(msg.Message.ID, 10, 0)
				if err != nil {
					db.log.Info("error", zap.Error(err))
					continue
				}
				msgts := ((msgid >> 22) + 1420070400000) / 1000

				dayAgo := ts.Unix() - int64((time.Hour * 24).Seconds())

				if msgts > dayAgo {
					messagelog = append(messagelog, msg)
				}
			}

		}
		return nil
	})
	return messagelog, err
}

func (db *DB) RunGC() error {
	fmt.Println("running gc")
	return db.db.RunValueLogGC(0.5)
}
