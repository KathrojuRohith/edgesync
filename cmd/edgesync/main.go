package main

import (
 "context"
 "flag"
 "log"
 "net/http"
 "os"
 "os/signal"
 "path/filepath"
 "syscall"
 "time"
 "edgesync/internal/web"
 "edgesync/pkg/kv"
 "edgesync/pkg/lsm"
)

func main() {
 addr:=flag.String("http",":8080","HTTP listen address")
 data:=flag.String("data","./data/solo","persistent data directory")
 seed:=flag.Bool("seed",false,"load demo logistics records")
 flag.Parse()
 engine,err:=lsm.Open(lsm.Options{Dir:filepath.Join(*data,"lsm"),SyncWrites:true})
 if err!=nil{log.Fatalf("open LSM engine: %v",err)}
 defer engine.Close()
 var store kv.Store=engine
 if *seed { web.Seed(store) }
 server:=&http.Server{Addr:*addr,Handler:web.NewServer(store),ReadHeaderTimeout:5*time.Second}
 ctx,stop:=signal.NotifyContext(context.Background(),os.Interrupt,syscall.SIGTERM);defer stop()
 go func(){log.Printf("EdgeSync web/API listening on %s",*addr);if err:=server.ListenAndServe();err!=nil&&err!=http.ErrServerClosed{log.Fatalf("HTTP server: %v",err)}}()
 <-ctx.Done()
 shutdown,cancel:=context.WithTimeout(context.Background(),5*time.Second);defer cancel()
 _=server.Shutdown(shutdown)
}
