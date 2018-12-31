package main

import (
	"sync"

	"github.com/bwmarrin/discordgo"
)

type MemberCache struct {
	sync.RWMutex
	storage map[string]*discordgo.Member
}

func NewMemCache() *MemberCache {
	return &MemberCache{
		storage: make(map[string]*discordgo.Member),
	}
}
func (m *MemberCache) Put(mem *discordgo.Member) {
	m.Lock()
	m.storage[mem.GuildID+mem.User.ID] = mem
	m.Unlock()
}
func (m *MemberCache) Get(key string) (mem *discordgo.Member, ok bool) {
	m.RLock()
	res, ok := m.storage[key]
	m.RUnlock()
	return res, ok
}
func (m *MemberCache) Delete(key string) {
	m.Lock()
	delete(m.storage, key)
	m.Unlock()
}
