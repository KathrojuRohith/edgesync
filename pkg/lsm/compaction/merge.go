// Package compaction merges sorted sources into one sorted stream.
package compaction
import("bytes";"edgesync/pkg/lsm/entry")
func Merge(srcs []entry.Iterator,dropTombstones bool,emit func(entry.Entry)error)error{heads:=make([]*entry.Entry,len(srcs));advance:=func(i int)error{e,ok,err:=srcs[i].Next();if err!=nil{return err};if ok{heads[i]=&e}else{heads[i]=nil};return nil};for i:=range srcs{if err:=advance(i);err!=nil{return err}};for{best:=-1;for i,h:=range heads{if h!=nil&&(best<0||bytes.Compare(h.Key,heads[best].Key)<0){best=i}};if best<0{return nil};winner:=*heads[best];for i:=range heads{if heads[i]!=nil&&bytes.Equal(heads[i].Key,winner.Key){if err:=advance(i);err!=nil{return err}}};if winner.Deleted&&dropTombstones{continue};if err:=emit(winner);err!=nil{return err}}}
