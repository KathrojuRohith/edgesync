package main

import (
 "bytes"
 "context"
 "io"
 "flag"
 "log"
 "net/http"
 "os"
 "os/signal"
 "path/filepath"
 "strings"
 "syscall"
 "time"
 "edgesync/internal/raft"
 "edgesync/internal/raftkv"
 "edgesync/internal/web"
 "edgesync/pkg/kv"
 "edgesync/pkg/lsm"
)

func proxyToLeader(next http.Handler,node *raft.Node,members map[string]string) http.Handler { return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request) { if node!=nil && strings.HasPrefix(r.URL.Path,"/api/") { st:=node.Status(); if st.LeaderID!="" && st.LeaderID!=st.ID { if addr,ok:=members[st.LeaderID];ok { body,err:=io.ReadAll(r.Body);if err!=nil{http.Error(w,err.Error(),400);return};req,err:=http.NewRequestWithContext(r.Context(),r.Method,"http://"+addr+r.URL.RequestURI(),bytes.NewReader(body));if err!=nil{http.Error(w,err.Error(),500);return};req.Header=r.Header.Clone();resp,err:=(&http.Client{Timeout:4*time.Second}).Do(req);if err!=nil{http.Error(w,"leader forwarding failed: "+err.Error(),502);return};defer resp.Body.Close();for k,vs:=range resp.Header{for _,v:=range vs{w.Header().Add(k,v)}};w.WriteHeader(resp.StatusCode);_,_=io.Copy(w,resp.Body);return } } };next.ServeHTTP(w,r) }) }

func main(){
 id:=flag.String("id","node-1","node identifier")
 addr:=flag.String("http",":8080","HTTP listen address")
 data:=flag.String("data","./data/solo","persistent data directory")
 peersText:=flag.String("peers","","cluster members id=host:port,...")
 seed:=flag.Bool("seed",false,"load sample logistics records in standalone mode")
 pprof:=flag.String("pprof","","optional pprof listen address")
 flag.Parse()
 if *pprof!=""{go func(){log.Printf("pprof listening on %s",*pprof);if e:=http.ListenAndServe(*pprof,nil);e!=nil{log.Printf("pprof stopped: %v",e)}}()}
 engine,err:=lsm.Open(lsm.Options{Dir:filepath.Join(*data,"lsm"),SyncWrites:true})
 if err!=nil{log.Fatalf("open LSM engine: %v",err)}
 var served kv.Store=engine
 mux:=http.NewServeMux();members:=map[string]string{};var raftNode *raft.Node
 if *peersText!=""{
  var ids []string
  for _,part:=range strings.Split(*peersText,","){pair:=strings.SplitN(strings.TrimSpace(part),"=",2);if len(pair)!=2||pair[0]==""||pair[1]==""{log.Fatalf("invalid peer %q; expected id=host:port",part)};members[pair[0]]=pair[1];ids=append(ids,pair[0])}
  if _,ok:=members[*id];!ok{log.Fatalf("node %q must appear in -peers",*id)}
  var other []string;for _,member:=range ids{if member!=*id{other=append(other,member)}}
  transport:=raft.NewHTTPTransport(*id,members)
  replicated:=raftkv.New(engine)
  node,e:=raft.New(raft.Config{ID:*id,Peers:other,Transport:transport,DataDir:filepath.Join(*data,"raft"),Apply:replicated.Apply})
  if e!=nil{log.Fatalf("start Raft: %v",e)}
  replicated.Attach(node);raftNode=node;served=replicated;mux.Handle("/raft/",transport.Handler(node))
 }else if *seed{web.Seed(served)}
 mux.Handle("/",proxyToLeader(web.NewServer(served),raftNode,members))
 server:=&http.Server{Addr:*addr,Handler:mux,ReadHeaderTimeout:5*time.Second}
 ctx,stop:=signal.NotifyContext(context.Background(),os.Interrupt,syscall.SIGTERM);defer stop()
 go func(){log.Printf("EdgeSync %s web/API listening on %s",*id,*addr);if e:=server.ListenAndServe();e!=nil&&e!=http.ErrServerClosed{log.Fatalf("HTTP server: %v",e)}}()
 <-ctx.Done()
 shutdown,cancel:=context.WithTimeout(context.Background(),5*time.Second);defer cancel();_=server.Shutdown(shutdown);_=served.Close()
}
