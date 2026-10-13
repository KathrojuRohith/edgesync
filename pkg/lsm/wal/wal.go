// Package wal implements a CRC-protected append-only write-ahead log.
package wal
import("encoding/binary";"hash/crc32";"os";"sync")
const(OpPut byte=1;OpDelete byte=2)
type Record struct{Op byte;Key []byte;Value []byte}
func Encode(r Record)[]byte{n:=1+4+len(r.Key)+len(r.Value);b:=make([]byte,8+n);binary.LittleEndian.PutUint32(b[4:8],uint32(n));b[8]=r.Op;binary.LittleEndian.PutUint32(b[9:13],uint32(len(r.Key)));copy(b[13:],r.Key);copy(b[13+len(r.Key):],r.Value);binary.LittleEndian.PutUint32(b[:4],crc32.ChecksumIEEE(b[4:]));return b}
func Decode(data []byte,fn func(Record))int{p:=0;for p+8<=len(data){n:=int(binary.LittleEndian.Uint32(data[p+4:p+8]));if n<5||p+8+n>len(data)||crc32.ChecksumIEEE(data[p+4:p+8+n])!=binary.LittleEndian.Uint32(data[p:p+4]){break};body:=data[p+8:p+8+n];k:=int(binary.LittleEndian.Uint32(body[1:5]));if 5+k>len(body){break};fn(Record{Op:body[0],Key:append([]byte(nil),body[5:5+k]...),Value:append([]byte(nil),body[5+k:]...)});p+=8+n};return p}
func Replay(path string,fn func(Record))error{b,e:=os.ReadFile(path);if os.IsNotExist(e){return nil};if e!=nil{return e};Decode(b,fn);return nil}
type Log struct{mu sync.Mutex;f *os.File;syncEach bool}
func Create(path string,syncEach bool)(*Log,error){f,e:=os.OpenFile(path,os.O_CREATE|os.O_WRONLY|os.O_APPEND,0644);if e!=nil{return nil,e};return &Log{f:f,syncEach:syncEach},nil}
func(l *Log)Append(r Record)error{b:=Encode(r);l.mu.Lock();defer l.mu.Unlock();if _,e:=l.f.Write(b);e!=nil{return e};if l.syncEach{return l.f.Sync()};return nil}
func(l *Log)Close()error{l.mu.Lock();defer l.mu.Unlock();a:=l.f.Sync();b:=l.f.Close();if a!=nil{return a};return b}
