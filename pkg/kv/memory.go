package kv

import ("bytes"; "sort"; "sync")

type MemoryStore struct { mu sync.RWMutex; data map[string][]byte; closed bool }
var _ Store = (*MemoryStore)(nil)
func NewMemoryStore() *MemoryStore { return &MemoryStore{data:make(map[string][]byte)} }
func (m *MemoryStore) Put(key,value []byte) error { if len(key)==0{return ErrEmptyKey}; m.mu.Lock(); defer m.mu.Unlock(); if m.closed{return ErrClosed}; m.data[string(key)]=clone(value); return nil }
func (m *MemoryStore) Get(key []byte) ([]byte,error) { m.mu.RLock(); defer m.mu.RUnlock(); if m.closed{return nil,ErrClosed}; v,ok:=m.data[string(key)]; if !ok{return nil,ErrNotFound}; return clone(v),nil }
func (m *MemoryStore) Delete(key []byte) error { m.mu.Lock(); defer m.mu.Unlock(); if m.closed{return ErrClosed}; delete(m.data,string(key)); return nil }
func (m *MemoryStore) Scan(prefix []byte,limit int)([]Pair,error){ m.mu.RLock(); defer m.mu.RUnlock(); if m.closed{return nil,ErrClosed}; keys:=make([]string,0,len(m.data)); for k:=range m.data {if bytes.HasPrefix([]byte(k),prefix){keys=append(keys,k)}}; sort.Strings(keys); if limit>0&&len(keys)>limit{keys=keys[:limit]}; out:=make([]Pair,0,len(keys)); for _,k:=range keys{out=append(out,Pair{Key:[]byte(k),Value:clone(m.data[k])})}; return out,nil }
func (m *MemoryStore) Close() error {m.mu.Lock();defer m.mu.Unlock();m.closed=true;m.data=nil;return nil}
func (m *MemoryStore) Len() int {m.mu.RLock();defer m.mu.RUnlock();return len(m.data)}
