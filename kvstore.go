package stare

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dgraph-io/badger"
	"github.com/dgraph-io/badger/options"
	"go.uber.org/zap"
)

type Store struct {
	db     *badger.DB
	logger *ZapLogger
}

func NewStore(logger *ZapLogger) (*Store, error) {
	logger = logger.Named("kvstore").(*ZapLogger)
	badgerLogger := logger.Named("badger").(*ZapLogger)
	s := &Store{
		logger: logger,
	}

	opts := badger.DefaultOptions("./data")
	opts.Truncate = true
	opts.ValueLogLoadingMode = options.FileIO
	opts.NumVersionsToKeep = 1
	opts.Logger = badgerLogger

	db, err := badger.Open(opts)
	if err != nil {
		s.logger.Info("error", zap.Error(err))
		return nil, err
	}
	s.db = db

	go s.RunGC()

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func encodeGob(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(v)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decodeGob(data []byte, v interface{}) error {
	buffer := bytes.NewReader(data)
	return gob.NewDecoder(buffer).Decode(v)
}

func (s *Store) SetMember(m *discordgo.Member) error {
	enc, err := encodeGob(m)
	if err != nil {
		s.logger.Error("failed to encode member", zap.Error(err))
		return err
	}

	key := fmt.Sprintf("member:%v:%v", m.GuildID, m.User.ID)
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), enc)
	})
}

func (s *Store) GetMember(gid, uid string) (*discordgo.Member, error) {
	var member discordgo.Member
	key := fmt.Sprintf("member:%v:%v", gid, uid)
	if err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		value, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}
		return decodeGob(value, &member)
	}); err != nil {
		if err != badger.ErrKeyNotFound {
			s.logger.Error("failed to read value", zap.Error(err))
		}
		return nil, err
	}

	return &member, nil
}

func (s *Store) DeleteMember(gid, uid string) error {
	key := fmt.Sprintf("member:%v:%v", gid, uid)
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

func (s *Store) SetMessage(msg *DiscordMessage) error {
	messageKey := fmt.Sprintf("message:%s:%s:%s", msg.Message.GuildID, msg.Message.ChannelID, msg.Message.ID)
	enc, err := encodeGob(msg)
	if err != nil {
		return fmt.Errorf("failed to encode DiscordMessage: %w", err)
	}

	indexKey := fmt.Sprintf("index:%s:%s:%s:%s", msg.Message.GuildID, msg.Message.Author.ID, msg.Message.Timestamp, msg.Message.ID)
	indexValue := messageKey

	return s.db.Update(func(txn *badger.Txn) error {
		entry := badger.NewEntry([]byte(messageKey), enc).WithTTL(24 * time.Hour)
		if err := txn.SetEntry(entry); err != nil {
			return err
		}

		indexEntry := badger.NewEntry([]byte(indexKey), []byte(indexValue)).WithTTL(24 * time.Hour)
		return txn.SetEntry(indexEntry)
	})
}

func (s *Store) GetMessage(gid, cid, mid string) (*DiscordMessage, error) {
	var message DiscordMessage
	key := fmt.Sprintf("message:%v:%v:%v", gid, cid, mid)
	if err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		value, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}
		return decodeGob(value, &message)
	}); err != nil {
		if err != badger.ErrKeyNotFound {
			s.logger.Error("failed to read message", zap.Error(err))
		}
		return nil, err
	}

	return &message, nil
}

func (s *Store) GetMessageLog(gid, uid string) ([]*DiscordMessage, error) {
	prefix := fmt.Sprintf("index:%v:%v:", gid, uid)
	var messages []*DiscordMessage
	err := s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Seek([]byte(prefix)); it.ValidForPrefix([]byte(prefix)); it.Next() {
			item := it.Item()
			var messageKey string
			value, err := item.ValueCopy(nil)
			if err != nil {
				s.logger.Error("failed to read value", zap.Error(err))
				continue
			}
			messageKey = string(value)

			var message DiscordMessage
			err = s.db.View(func(txn *badger.Txn) error {
				item, err := txn.Get([]byte(messageKey))
				if err != nil {
					return err
				}
				value, err := item.ValueCopy(nil)
				if err != nil {
					return err
				}
				return decodeGob(value, &message)
			})
			if err != nil {
				s.logger.Error("failed to read message", zap.Error(err))
				continue
			}

			messages = append(messages, &message)
		}
		return nil
	})
	if err != nil {
		s.logger.Error("failed to read messages", zap.Error(err))
		return nil, err
	}

	return messages, err
}

func (s *Store) RunGC() {
	gcTicker := time.NewTicker(time.Hour)
	for range gcTicker.C {
		for {
			err := s.db.RunValueLogGC(0.7)
			if err == badger.ErrNoRewrite {
				break
			}
		}
	}
}
