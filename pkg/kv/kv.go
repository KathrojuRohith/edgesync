package kv

import "errors"

var ErrNotFound = errors.New("key not found")

type Store interface {
	Put(key, value string) error
	Get(key string) (string, error)
	Delete(key string) error
	Scan(prefix string) (map[string]string, error)
	Close() error
}
