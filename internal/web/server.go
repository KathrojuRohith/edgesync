package web

import (
 "encoding/json"
 "errors"
 "fmt"
 "net/http"
 "os"
 "strings"
 "sync/atomic"
 "time"
 "edgesync/pkg/kv"
)

type Server struct { store kv.Store; started time.Time; requests atomic.Uint64 }
func NewServer(store kv.Store) http.Handler {
 s:=&Server{store:store,started:time.Now()}
 mux:=http.NewServeMux()
 mux.HandleFunc("/",s.home)
 mux.HandleFunc("/healthz",func(w http.ResponseWriter,r *http.Request){writeJSON(w,200,map[string]any{"status":"ok","uptime_seconds":int(time.Since(s.started).Seconds())})})
 mux.HandleFunc("/metrics",s.metrics)
 mux.HandleFunc("/api/kv",s.kvHandler)
 mux.HandleFunc("/api/scan",s.scan)
 return mux
}
func(s *Server)home(w http.ResponseWriter,r *http.Request){
 if r.URL.Path!="/" {http.NotFound(w,r);return}
 page,err:=os.ReadFile("web/index.html")
 if err==nil&&len(page)>0 {w.Header().Set("Content-Type","text/html; charset=utf-8");_,_=w.Write(page);return}
 w.Header().Set("Content-Type","text/html; charset=utf-8")
 _,_=fmt.Fprint(w,"<!doctype html><html><head><title>EdgeSync</title></head><body><h1>EdgeSync Console</h1><p>Storage API is online.</p><a href=\"/healthz\">Health</a> · <a href=\"/metrics\">Metrics</a></body></html>")
}
type mutation struct {Key string;Value string}
type requestIDStore interface { PutID(string, []byte, []byte) error; DeleteID(string, []byte) error }
func(s *Server)kvHandler(w http.ResponseWriter,r *http.Request){
 s.requests.Add(1)
 switch r.Method {
 case http.MethodPut:
  var m mutation
  if err:=json.NewDecoder(http.MaxBytesReader(w,r.Body,1<<20)).Decode(&m);err!=nil{writeErr(w,400,err);return}
  if strings.TrimSpace(m.Key)==""{writeErr(w,400,kv.ErrEmptyKey);return}
  var putErr error; if ids,ok:=s.store.(requestIDStore);ok&&r.Header.Get("X-Request-ID")!="" { putErr=ids.PutID(r.Header.Get("X-Request-ID"),[]byte(m.Key),[]byte(m.Value)) } else { putErr=s.store.Put([]byte(m.Key),[]byte(m.Value)) }; if putErr!=nil{writeErr(w,500,putErr);return}
  writeJSON(w,200,map[string]any{"ok":true,"key":m.Key})
 case http.MethodGet:
  key:=r.URL.Query().Get("key");value,err:=s.store.Get([]byte(key))
  if errors.Is(err,kv.ErrNotFound){writeErr(w,404,err);return};if err!=nil{writeErr(w,500,err);return}
  writeJSON(w,200,map[string]string{"key":key,"value":string(value)})
 case http.MethodDelete:
  key:=r.URL.Query().Get("key");if key==""{writeErr(w,400,kv.ErrEmptyKey);return}
  var delErr error; if ids,ok:=s.store.(requestIDStore);ok&&r.Header.Get("X-Request-ID")!="" { delErr=ids.DeleteID(r.Header.Get("X-Request-ID"),[]byte(key)) } else { delErr=s.store.Delete([]byte(key)) }; if delErr!=nil{writeErr(w,500,delErr);return};writeJSON(w,200,map[string]any{"ok":true,"key":key})
 default:w.Header().Set("Allow","GET, PUT, DELETE");writeErr(w,405,errors.New("method not allowed"))
 }
}
func(s *Server)scan(w http.ResponseWriter,r *http.Request){
 if r.Method!=http.MethodGet{writeErr(w,405,errors.New("method not allowed"));return}
 pairs,err:=s.store.Scan([]byte(r.URL.Query().Get("prefix")),1000);if err!=nil{writeErr(w,500,err);return}
 result:=make(map[string]string,len(pairs));for _,p:=range pairs{result[string(p.Key)]=string(p.Value)};writeJSON(w,200,result)
}
func(s *Server)metrics(w http.ResponseWriter,r *http.Request){
 w.Header().Set("Content-Type","text/plain; version=0.0.4")
 fmt.Fprintln(w,"# HELP edgesync_http_requests_total Total HTTP API requests")
 fmt.Fprintln(w,"# TYPE edgesync_http_requests_total counter")
 fmt.Fprintf(w,"edgesync_http_requests_total %d\n",s.requests.Load())
 fmt.Fprintln(w,"# TYPE edgesync_uptime_seconds gauge")
 fmt.Fprintf(w,"edgesync_uptime_seconds %d\n",int(time.Since(s.started).Seconds()))
}
func writeJSON(w http.ResponseWriter,status int,v any){w.Header().Set("Content-Type","application/json");w.WriteHeader(status);_=json.NewEncoder(w).Encode(v)}
func writeErr(w http.ResponseWriter,status int,e error){writeJSON(w,status,map[string]string{"error":e.Error()})}
func Seed(s kv.Store){
 records:=map[string]string{
 "driver:DRV501":"{\"name\":\"Ravi\",\"status\":\"BUSY\",\"vehicle\":\"BIKE\",\"location\":\"Vijayawada\"}",
 "driver:DRV502":"{\"name\":\"Lakshmi\",\"status\":\"BUSY\",\"vehicle\":\"VAN\",\"location\":\"Guntur\"}",
 "driver:DRV503":"{\"name\":\"Kiran\",\"status\":\"AVAILABLE\",\"vehicle\":\"VAN\",\"location\":\"Tenali\"}",
 "order:ORD1001":"{\"customer\":\"C100\",\"driver\":\"DRV501\",\"source\":\"Vijayawada\",\"destination\":\"Guntur\",\"status\":\"IN_TRANSIT\",\"priority\":\"HIGH\"}",
 "order:ORD1002":"{\"customer\":\"C101\",\"driver\":\"DRV502\",\"source\":\"Guntur\",\"destination\":\"Tenali\",\"status\":\"PICKED_UP\",\"priority\":\"NORMAL\"}",
 "order:ORD1003":"{\"customer\":\"C102\",\"driver\":null,\"source\":\"Tenali\",\"destination\":\"Vijayawada\",\"status\":\"CREATED\",\"priority\":\"NORMAL\"}",
 }
 for k,v:=range records{_=s.Put([]byte(k),[]byte(v))}
}
