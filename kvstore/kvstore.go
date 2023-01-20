package kvstore

import (
	"bytes"
	"encoding/gob"
	"fmt"
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
		s.log.Error("failed to encode member", zap.Error(err))
		return err
	}

	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(fmt.Sprintf("member:%v:%v", m.GuildID, m.User.ID)), buf.Bytes())
	})
}

func (s *Store) GetMember(gid, uid string) (*discordgo.Member, error) {
	var body []byte
	if err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(fmt.Sprintf("member:%v:%v", gid, uid)))
		if err != nil {
			return err
		}
		body, err = item.ValueCopy(nil)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		s.log.Error("failed to read value", zap.Error(err))
		return nil, err
	}

	mem := &discordgo.Member{}
	err := gob.NewDecoder(bytes.NewReader(body)).Decode(mem)
	if err != nil {
		s.log.Error("failed to decode member", zap.Error(err))
		return nil, err
	}
	return mem, nil
}

func (s *Store) DeleteMember(gid, uid string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(fmt.Sprintf("member:%v:%v", gid, uid)))
	})
}

func (s *Store) SetMessage(msg *DiscordMessage) error {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(msg)
	if err != nil {
		s.log.Error("failed to encode message", zap.Error(err))
		return err
	}

	err = s.db.Update(func(txn *badger.Txn) error {
		e := &badger.Entry{
			Key:       []byte(fmt.Sprintf("message:%v:%v:%v", msg.Message.GuildID, msg.Message.ChannelID, msg.Message.ID)),
			Value:     buf.Bytes(),
			ExpiresAt: uint64(time.Now().Add(time.Hour * 24).Unix()),
		}
		return txn.SetEntry(e)
	})
	return err
}

func (s *Store) GetMessage(gid, cid, mid string) (*DiscordMessage, error) {
	var body []byte
	if err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(fmt.Sprintf("message:%v:%v:%v", gid, cid, mid)))
		if err != nil {
			return err
		}
		body, err = item.ValueCopy(nil)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		s.log.Error("failed to read message", zap.Error(err))
		return nil, err
	}

	msg := DiscordMessage{}
	err := gob.NewDecoder(bytes.NewReader(body)).Decode(&msg)
	if err != nil {
		s.log.Error("failed to decode message", zap.Error(err))
		return nil, err
	}
	return &msg, nil
}

func (s *Store) GetMessageLog(gid, cid, uid string) ([]*DiscordMessage, error) {
	var messages []*DiscordMessage
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()

		// surely this can be changed to msg:guild:user
		// this means the msg key needs to be changed as well
		// probably to msg:guild:user:channel:msgID?
		prefix := []byte(fmt.Sprintf("message:%v:%v", gid, cid))
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()

			body, err := item.ValueCopy(nil)
			if err != nil {
				s.log.Error("error", zap.Error(err))
				continue
			}
			msg := DiscordMessage{}
			err = gob.NewDecoder(bytes.NewReader(body)).Decode(&msg)
			if err != nil {
				s.log.Error("error", zap.Error(err))
				continue
			}

			if msg.Message.Author.ID == uid {
				ts, err := ParseSnowflake(msg.Message.ID)
				if err != nil {
					continue
				}

				if time.Since(ts) < time.Hour*24 {
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

func ParseSnowflake(id string) (time.Time, error) {
	n, err := strconv.ParseInt(id, 0, 63)
	if err != nil {
		return time.Now(), err
	}
	return time.Unix(((n>>22)+1420070400000)/1000, 0), nil
}
