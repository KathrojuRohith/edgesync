// Package lsm is the LSM-tree storage engine: WAL + memtable + SSTables + full compaction.
package lsm

import (
 "bytes"
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "sort"
 "strconv"
 "strings"
 "sync"
 "edgesync/pkg/kv"
 "edgesync/pkg/lsm/compaction"
 "edgesync/pkg/lsm/entry"
 "edgesync/pkg/lsm/memtable"
 "edgesync/pkg/lsm/sstable"
 "edgesync/pkg/lsm/wal"
)
type Options struct { Dir string; MemtableBytes int; SyncWrites bool; CompactAt int }
type Stats struct { MemtableBytes int; Tables int; Flushes uint64; Compactions uint64 }
type handle struct { seq uint64; path string; t *sstable.Table }
type Engine struct { mu sync.RWMutex; opts Options; mem *memtable.Table; log *wal.Log; walPath string; tables []*handle; next uint64; closed bool; flushes uint64; compactions uint64 }
var _ kv.Store = (*Engine)(nil)
type fileRef struct { seq uint64; path string }
func parseSeq(name,prefix,suffix string)(uint64,bool){if !strings.HasPrefix(name,prefix)||!strings.HasSuffix(name,suffix){return 0,false};n,e:=strconv.ParseUint(strings.TrimSuffix(strings.TrimPrefix(name,prefix),suffix),10,64);return n,e==nil}
func Open(opts Options)(*Engine,error){
 if opts.MemtableBytes<=0{opts.MemtableBytes=4<<20};if opts.CompactAt<=0{opts.CompactAt=4};if err:=os.MkdirAll(opts.Dir,0755);err!=nil{return nil,err}
 e:=&Engine{opts:opts,mem:memtable.New()};ents,err:=os.ReadDir(opts.Dir);if err!=nil{return nil,err};var tabs,wals []fileRef;var maxSeq,maxCmp uint64
 for _,d:=range ents{name:=d.Name();path:=filepath.Join(opts.Dir,name);if strings.HasSuffix(name,".tmp"){_=os.Remove(path);continue};if s,ok:=parseSeq(name,"sst-",".sst");ok{tabs=append(tabs,fileRef{s,path});if s>maxSeq{maxSeq=s}}else if s,ok:=parseSeq(name,"cmp-",".sst");ok{tabs=append(tabs,fileRef{s,path});if s>maxSeq{maxSeq=s};if s>maxCmp{maxCmp=s}}else if s,ok:=parseSeq(name,"wal-",".log");ok{wals=append(wals,fileRef{s,path});if s>maxSeq{maxSeq=s}}}
 e.next=maxSeq+1;sort.Slice(tabs,func(i,j int)bool{return tabs[i].seq<tabs[j].seq});sort.Slice(wals,func(i,j int)bool{return wals[i].seq<wals[j].seq})
 for _,tf:=range tabs{if tf.seq<maxCmp{_=os.Remove(tf.path);continue};t,er:=sstable.Open(tf.path);if er!=nil{return nil,fmt.Errorf("open %s: %w",tf.path,er)};e.tables=append(e.tables,&handle{seq:tf.seq,path:tf.path,t:t})}
 for _,w:=range wals{er:=wal.Replay(w.path,func(r wal.Record){if r.Op==wal.OpDelete{e.mem.Delete(r.Key)}else{e.mem.Put(r.Key,r.Value)}});if er!=nil{return nil,er}}
 if e.mem.Len()>0{h,er:=e.writeTable("sst",[]entry.Iterator{entry.NewSliceIterator(e.mem.Entries())},e.mem.Len(),false);if er!=nil{return nil,er};e.tables=append(e.tables,h);e.flushes++;e.mem=memtable.New()}
 for _,w:=range wals{_=os.Remove(w.path)};if err:=e.newWAL();err!=nil{return nil,err};return e,nil
}
func(e *Engine)newWAL()error{seq:=e.next;e.next++;e.walPath=filepath.Join(e.opts.Dir,fmt.Sprintf("wal-%08d.log",seq));l,err:=wal.Create(e.walPath,e.opts.SyncWrites);if err!=nil{return err};e.log=l;return nil}
func syncDir(dir string){if d,err:=os.Open(dir);err==nil{_=d.Sync();_=d.Close()}}
func(e *Engine)writeTable(prefix string,srcs []entry.Iterator,expected int,dropTombstones bool)(*handle,error){seq:=e.next;e.next++;final:=filepath.Join(e.opts.Dir,fmt.Sprintf("%s-%08d.sst",prefix,seq));tmp:=final+".tmp";w,err:=sstable.NewWriter(tmp,expected);if err!=nil{return nil,err};if err:=compaction.Merge(srcs,dropTombstones,w.Add);err!=nil{w.Abort();_=os.Remove(tmp);return nil,err};if err:=w.Finish();err!=nil{_=os.Remove(tmp);return nil,err};if err:=os.Rename(tmp,final);err!=nil{return nil,err};syncDir(e.opts.Dir);t,err:=sstable.Open(final);if err!=nil{return nil,err};return &handle{seq:seq,path:final,t:t},nil}
func(e *Engine)write(op byte,key,value []byte)error{if len(key)==0{return kv.ErrEmptyKey};e.mu.Lock();defer e.mu.Unlock();if e.closed{return kv.ErrClosed};if err:=e.log.Append(wal.Record{Op:op,Key:key,Value:value});err!=nil{return err};if op==wal.OpPut{e.mem.Put(key,value)}else{e.mem.Delete(key)};if e.mem.Size()>=e.opts.MemtableBytes{return e.flushLocked()};return nil}
func(e *Engine)Put(key,value []byte)error{return e.write(wal.OpPut,key,value)}
func(e *Engine)Delete(key []byte)error{return e.write(wal.OpDelete,key,nil)}
func(e *Engine)Get(key []byte)([]byte,error){e.mu.RLock();defer e.mu.RUnlock();if e.closed{return nil,kv.ErrClosed};if v,found,deleted:=e.mem.Get(key);found{if deleted{return nil,kv.ErrNotFound};return v,nil};for i:=len(e.tables)-1;i>=0;i--{v,found,deleted,err:=e.tables[i].t.Get(key);if err!=nil{return nil,err};if found{if deleted{return nil,kv.ErrNotFound};return v,nil}};return nil,kv.ErrNotFound}
var errStop=errors.New("stop")
func(e *Engine)Scan(prefix []byte,limit int)([]kv.Pair,error){e.mu.RLock();defer e.mu.RUnlock();if e.closed{return nil,kv.ErrClosed};srcs:=[]entry.Iterator{entry.NewSliceIterator(e.mem.Entries())};for i:=len(e.tables)-1;i>=0;i--{srcs=append(srcs,e.tables[i].t.Iter())};var out []kv.Pair;err:=compaction.Merge(srcs,true,func(en entry.Entry)error{if bytes.HasPrefix(en.Key,prefix){out=append(out,kv.Pair{Key:append([]byte(nil),en.Key...),Value:append([]byte(nil),en.Value...)});if limit>0&&len(out)>=limit{return errStop}}else if bytes.Compare(en.Key,prefix)>0{return errStop};return nil});if err!=nil&&err!=errStop{return nil,err};return out,nil}
func(e *Engine)flushLocked()error{ents:=e.mem.Entries();if len(ents)==0{return nil};h,err:=e.writeTable("sst",[]entry.Iterator{entry.NewSliceIterator(ents)},len(ents),false);if err!=nil{return err};e.tables=append(e.tables,h);e.flushes++;old:=e.walPath;if err:=e.log.Close();err!=nil{return err};_=os.Remove(old);e.mem=memtable.New();if err:=e.newWAL();err!=nil{return err};if len(e.tables)>=e.opts.CompactAt{return e.compactLocked()};return nil}
func(e *Engine)compactLocked()error{srcs:=make([]entry.Iterator,0,len(e.tables));expected:=0;for i:=len(e.tables)-1;i>=0;i--{srcs=append(srcs,e.tables[i].t.Iter());expected+=int(e.tables[i].t.Count())};h,err:=e.writeTable("cmp",srcs,expected,true);if err!=nil{return err};old:=e.tables;e.tables=[]*handle{h};for _,o:=range old{_=o.t.Close();_=os.Remove(o.path)};e.compactions++;return nil}
func(e *Engine)Stats()Stats{e.mu.RLock();defer e.mu.RUnlock();return Stats{MemtableBytes:e.mem.Size(),Tables:len(e.tables),Flushes:e.flushes,Compactions:e.compactions}}
func(e *Engine)Close()error{e.mu.Lock();defer e.mu.Unlock();if e.closed{return nil};e.closed=true;err:=e.log.Close();for _,h:=range e.tables{_=h.t.Close()};return err}
