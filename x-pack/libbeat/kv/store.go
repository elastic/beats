package kv

import "time"

type Kv interface {
	Connect() error
	Get([]byte) ([]byte, error)
	Set([]byte, []byte, time.Duration) error
	Close() error
}
