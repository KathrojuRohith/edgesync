// Package entry holds types shared by every LSM component.
package entry
type Entry struct { Key []byte; Value []byte; Deleted bool }
type Iterator interface { Next() (Entry,bool,error) }
type SliceIterator struct { es []Entry; i int }
func NewSliceIterator(es []Entry)*SliceIterator{return &SliceIterator{es:es}}
func(s *SliceIterator)Next()(Entry,bool,error){if s.i>=len(s.es){return Entry{},false,nil};e:=s.es[s.i];s.i++;return e,true,nil}
