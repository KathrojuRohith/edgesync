// Package memtable is the concurrent in-memory sorted write buffer.
package memtable

import (
 "bytes"
 "math/rand"
 "sync"
 "edgesync/pkg/lsm/entry"
)
const maxLevel=16
type node struct{e entry.Entry;next []*node}
type Table struct{mu sync.RWMutex;head *node;level int;bytes int;count int;rnd *rand.Rand}
func New()*Table{return &Table{head:&node{next:make([]*node,maxLevel)},level:1,rnd:rand.New(rand.NewSource(1))}}
func clone(b []byte)[]byte{if b==nil{return nil};return append([]byte{},b...)}
func(t *Table)randomLevel()int{l:=1;for l<maxLevel&&t.rnd.Intn(4)==0{l++};return l}
func(t *Table)set(key,value []byte,deleted bool){t.mu.Lock();defer t.mu.Unlock();var update [maxLevel]*node;x:=t.head;for i:=t.level-1;i>=0;i--{for x.next[i]!=nil&&bytes.Compare(x.next[i].e.Key,key)<0{x=x.next[i]};update[i]=x};if nx:=x.next[0];nx!=nil&&bytes.Equal(nx.e.Key,key){t.bytes+=len(value)-len(nx.e.Value);nx.e.Value=clone(value);nx.e.Deleted=deleted;return};lvl:=t.randomLevel();if lvl>t.level{for i:=t.level;i<lvl;i++{update[i]=t.head};t.level=lvl};n:=&node{e:entry.Entry{Key:clone(key),Value:clone(value),Deleted:deleted},next:make([]*node,lvl)};for i:=0;i<lvl;i++{n.next[i]=update[i].next[i];update[i].next[i]=n};t.bytes+=len(key)+len(value)+32;t.count++}
func(t *Table)Put(k,v []byte){t.set(k,v,false)}
func(t *Table)Delete(k []byte){t.set(k,nil,true)}
func(t *Table)Get(key []byte)([]byte,bool,bool){t.mu.RLock();defer t.mu.RUnlock();x:=t.head;for i:=t.level-1;i>=0;i--{for x.next[i]!=nil&&bytes.Compare(x.next[i].e.Key,key)<0{x=x.next[i]}};if nx:=x.next[0];nx!=nil&&bytes.Equal(nx.e.Key,key){if nx.e.Deleted{return nil,true,true};return clone(nx.e.Value),true,false};return nil,false,false}
func(t *Table)Size()int{t.mu.RLock();defer t.mu.RUnlock();return t.bytes}
func(t *Table)Len()int{t.mu.RLock();defer t.mu.RUnlock();return t.count}
func(t *Table)Entries()[]entry.Entry{t.mu.RLock();defer t.mu.RUnlock();out:=make([]entry.Entry,0,t.count);for x:=t.head.next[0];x!=nil;x=x.next[0]{e:=x.e;e.Key=clone(e.Key);e.Value=clone(e.Value);out=append(out,e)};return out}
