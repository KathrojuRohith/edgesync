package raftkv

import(
 "encoding/json"
 "sync"
 "time"
 "edgesync/internal/raft"
 "edgesync/pkg/kv"
)
type command struct{Op string `json:"op"`;Key []byte `json:"key,omitempty"`;Value []byte `json:"value,omitempty"`;ReqID string `json:"req,omitempty"`}
type Store struct{node *raft.Node;engine kv.Store;timeout time.Duration;mu sync.Mutex;seen map[string]struct{};order []string}
var _ kv.Store=(*Store)(nil)
func New(engine kv.Store)*Store{return &Store{engine:engine,timeout:3*time.Second,seen:map[string]struct{}{}}}
func(s *Store)Attach(n *raft.Node){s.node=n}
func(s *Store)Apply(_ uint64,data []byte)any{var c command;if e:=json.Unmarshal(data,&c);e!=nil{return e};if c.ReqID!=""{s.mu.Lock();_,dup:=s.seen[c.ReqID];s.mu.Unlock();if dup{return nil}};var err error;switch c.Op{case "put":err=s.engine.Put(c.Key,c.Value);case "del":err=s.engine.Delete(c.Key)};if err==nil&&c.ReqID!=""{s.mu.Lock();s.seen[c.ReqID]=struct{}{};s.order=append(s.order,c.ReqID);if len(s.order)>100000{delete(s.seen,s.order[0]);s.order=s.order[1:]};s.mu.Unlock()};return err}
func(s *Store)propose(c command)error{if s.node==nil{return s.engine.Put(c.Key,c.Value)};b,e:=json.Marshal(c);if e!=nil{return e};res,e:=s.node.Propose(b,s.timeout);if e!=nil{return e};if x,ok:=res.(error);ok&&x!=nil{return x};return nil}
func(s *Store)Put(key,value []byte)error{if len(key)==0{return kv.ErrEmptyKey};return s.propose(command{Op:"put",Key:key,Value:value})}
func(s *Store)Delete(key []byte)error{if len(key)==0{return kv.ErrEmptyKey};return s.propose(command{Op:"del",Key:key})}
func(s *Store)barrier()error{if s.node==nil{return nil};_,e:=s.node.Propose(nil,s.timeout);return e}
func(s *Store)Get(key []byte)([]byte,error){if e:=s.barrier();e!=nil{return nil,e};return s.engine.Get(key)}
func(s *Store)Scan(prefix []byte,limit int)([]kv.Pair,error){if e:=s.barrier();e!=nil{return nil,e};return s.engine.Scan(prefix,limit)}
func(s *Store)Close()error{if s.node!=nil{s.node.Stop()};return s.engine.Close()}
