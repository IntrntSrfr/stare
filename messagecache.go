package main

import (
	"sync"

	"github.com/bwmarrin/discordgo"
)

type DiscMessage struct {
	Message    *discordgo.Message
	Attachment [][]byte
}

type MessageCache struct {
	sync.RWMutex
	storage map[string]*DiscMessage
}

func NewMsgCache() *MessageCache {
	return &MessageCache{
		storage: make(map[string]*DiscMessage),
	}
}
func (m *MessageCache) Put(msg *DiscMessage) {
	m.Lock()
	m.storage[msg.Message.ID] = msg
	m.Unlock()
}
func (m *MessageCache) Update(msg *DiscMessage) {
	m.Lock()
	m.storage[msg.Message.ID].Message.Content = msg.Message.Content
	m.Unlock()
}
func (m *MessageCache) Get(key string) (msg *DiscMessage, ok bool) {
	m.RLock()
	res, ok := m.storage[key]
	m.RUnlock()
	return res, ok
}
func (m *MessageCache) Delete(key string) {
	m.Lock()
	delete(m.storage, key)
	m.Unlock()
}
func (m *MessageCache) Clear() {
	for _, val := range m.storage {
		m.Delete(val.Message.ID)
	}
}

type ByID []*DiscMessage

func (m ByID) Len() int {
	return len(m)
}

func (m ByID) Less(i, j int) bool {
	return m[i].Message.ID < m[j].Message.ID
}

func (m ByID) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}
