package kvstore

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/dgraph-io/badger/options"

	"github.com/bwmarrin/discordgo"
	"github.com/dgraph-io/badger"
)

type Store struct {
	db  *badger.DB
	log *zap.Logger
}

func NewStore(log *zap.Logger) (*Store, error) {
	s := &Store{
		log: log,
	}

	err := os.Mkdir("./data", 0740)
	if err != nil {
		return nil, err
	}

	opts := badger.DefaultOptions("./data")
	opts.Truncate = true
	opts.ValueLogLoadingMode = options.FileIO
	opts.NumVersionsToKeep = 1
	db, err := badger.Open(opts)
	if err != nil {
		s.log.Info("error", zap.Error(err))
		return nil, err
	}
	s.db = db

	go func(s *Store) {
		gcTimer := time.NewTicker(time.Hour)
		for range gcTimer.C {
			err := s.db.RunValueLogGC(0.5)
			if err != nil {
				s.log.Error("failed to run gc", zap.Error(err))
			}
		}
	}(s)

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) SetMember(m *discordgo.Member) error {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(m)
	if err != nil {
		s.log.Info("error", zap.Error(err))
		return err
	}

	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(fmt.Sprintf("member:%v:%v", m.GuildID, m.User.ID)), buf.Bytes())
	})
}

func (s *Store) GetMember(key string) (*discordgo.Member, error) {
	var body []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("member:" + key))
		if err != nil {
			s.log.Info("failed to lookup member", zap.Error(err))
			return err
		}
		body, err = item.ValueCopy(nil)
		if err != nil {
			s.log.Info("error", zap.Error(err))
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
		s.log.Info("error", zap.Error(err))
		return nil, err
	}
	return msg, nil
}

func (s *Store) DeleteMember(key string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte("member:" + key))
	})
}

func (s *Store) SetMessage(msg *DiscordMessage) error {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(msg)
	if err != nil {
		s.log.Info("failed to encode message", zap.Error(err))
		return err
	}

	err = s.db.Update(func(txn *badger.Txn) error {
		e := badger.Entry{
			Key:       []byte(fmt.Sprintf("message:%v:%v:%v", msg.Message.GuildID, msg.Message.ChannelID, msg.Message.ID)),
			Value:     buf.Bytes(),
			ExpiresAt: uint64(time.Now().Add(time.Hour * 24).Unix()),
		}
		return txn.SetEntry(&e)
	})
	return err
}

func (s *Store) GetMessage(key string) (*DiscordMessage, error) {
	var body []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("message:" + key))
		if err != nil {
			s.log.Info("failed to lookup message", zap.Error(err))
			return err
		}
		body, err = item.ValueCopy(nil)
		if err != nil {
			s.log.Info("error", zap.Error(err))
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	msg := DiscordMessage{}
	err = gob.NewDecoder(bytes.NewReader(body)).Decode(&msg)
	if err != nil {
		s.log.Info("failed to decode message", zap.Error(err))
		return nil, err
	}
	return &msg, nil
}

func (s *Store) GetMessageLog(m *discordgo.GuildBanAdd) ([]*DiscordMessage, error) {
	var messages []*DiscordMessage
	err := s.db.View(func(txn *badger.Txn) error {
		ts := time.Now()
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		// surely this can be changed to msg:guild:user
		// this means the msg key needs to be changed as well
		// probably to msg:guild:user:channel:msgID?
		prefix := []byte(fmt.Sprintf("message:%v:", m.GuildID))
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()

			body, err := item.ValueCopy(nil)
			if err != nil {
				s.log.Info("error", zap.Error(err))
				continue
			}
			msg := DiscordMessage{}
			err = gob.NewDecoder(bytes.NewReader(body)).Decode(&msg)
			if err != nil {
				s.log.Info("error", zap.Error(err))
				continue
			}

			if msg.Message.Author.ID == m.User.ID {

				msgid, err := strconv.ParseInt(msg.Message.ID, 10, 0)
				if err != nil {
					s.log.Info("error", zap.Error(err))
					continue
				}
				msgts := ((msgid >> 22) + 1420070400000) / 1000

				dayAgo := ts.Unix() - int64((time.Hour * 24).Seconds())

				if msgts > dayAgo {
					messages = append(messages, &msg)
				}
			}

		}
		return nil
	})
	return messages, err
}

func (s *Store) RunGC() error {
	fmt.Println("running gc")
	return s.db.RunValueLogGC(0.5)
}
