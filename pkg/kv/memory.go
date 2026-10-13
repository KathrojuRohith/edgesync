package kv

import "sync"

type MemoryStore struct {
	mu sync.RWMutex
	data map[string]string
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{data: make(map[string]string)} }
func (m *MemoryStore) Put(k, v string) error { m.mu.Lock(); defer m.mu.Unlock(); m.data[k] = v; return nil }
func (m *MemoryStore) Get(k string) (string, error) { m.mu.RLock(); defer m.mu.RUnlock(); v, ok := m.data[k]; if !ok { return "", ErrNotFound }; return v, nil }
func (m *MemoryStore) Delete(k string) error { m.mu.Lock(); defer m.mu.Unlock(); if _, ok := m.data[k]; !ok { return ErrNotFound }; delete(m.data, k); return nil }
func (m *MemoryStore) Scan(prefix string) (map[string]string, error) { m.mu.RLock(); defer m.mu.RUnlock(); out := map[string]string{}; for k,v := range m.data { if len(k) >= len(prefix) && k[:len(prefix)] == prefix { out[k] = v } }; return out, nil }
func (m *MemoryStore) Close() error { return nil }
