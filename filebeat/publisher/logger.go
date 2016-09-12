package publisher

import (
	"sync"

	"github.com/elastic/beats/filebeat/input"
)

type Log struct {
	wg *sync.WaitGroup
}

func NewLog(wg *sync.WaitGroup) *Log {
	return &Log{wg}
}

func (l *Log) Log(events []*input.Event) bool {
	for range events {
		l.wg.Done()
	}

	return true
}
