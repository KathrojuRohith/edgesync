// Package kv defines the storage contract used by every EdgeSync layer.
package kv

import "errors"

var ErrNotFound = errors.New("edgesync: key not found")
var ErrClosed = errors.New("edgesync: store is closed")
var ErrEmptyKey = errors.New("edgesync: empty key")

type Pair struct { Key []byte; Value []byte }
type Store interface {
 Put(key, value []byte) error
 Get(key []byte) ([]byte, error)
 Delete(key []byte) error
 Scan(prefix []byte, limit int) ([]Pair, error)
 Close() error
}
func clone(b []byte) []byte { if b == nil { return nil }; out:=make([]byte,len(b)); copy(out,b); return out }
