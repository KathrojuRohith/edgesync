// Package bloom is a compact Bloom filter.
package bloom
import("errors";"hash/fnv")
const(bitsPerKey=10;numHashes=7)
type Filter struct{bits []byte;k byte}
func New(n int)*Filter{nb:=n*bitsPerKey;if nb<64{nb=64};return &Filter{bits:make([]byte,(nb+7)/8),k:numHashes}}
func hash2(key []byte)(uint64,uint64){h:=fnv.New64a();_,_=h.Write(key);a:=h.Sum64();b:=(a*0x9E3779B97F4A7C15^(a>>29))|1;return a,b}
func(f *Filter)Add(key []byte){h1,h2:=hash2(key);nb:=uint64(len(f.bits))*8;for i:=uint64(0);i<uint64(f.k);i++{p:=(h1+i*h2)%nb;f.bits[p/8]|=1<<(p%8)}}
func(f *Filter)MayContain(key []byte)bool{h1,h2:=hash2(key);nb:=uint64(len(f.bits))*8;for i:=uint64(0);i<uint64(f.k);i++{p:=(h1+i*h2)%nb;if f.bits[p/8]&(1<<(p%8))==0{return false}};return true}
func(f *Filter)Marshal()[]byte{return append([]byte{f.k},f.bits...)}
func Unmarshal(b []byte)(*Filter,error){if len(b)<2{return nil,errors.New("bloom: too short")};return &Filter{k:b[0],bits:append([]byte(nil),b[1:]...)},nil}
