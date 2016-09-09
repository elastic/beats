package spooler

import (
	"sync"
	"sync/atomic"

	"github.com/elastic/beats/filebeat/input"
)

type Output struct {
	wg      *sync.WaitGroup
	done    <-chan struct{}
	spooler *Spooler
	isOpen  int32 // atomic indicator
}

func NewOutput(
	done <-chan struct{},
	s *Spooler,
	wg *sync.WaitGroup,
) *Output {
	return &Output{
		done:    done,
		spooler: s,
		wg:      wg,
		isOpen:  1,
	}
}

func (o *Output) Send(event *input.Event) bool {
	open := atomic.LoadInt32(&o.isOpen) == 1
	if !open {
		return false
	}

	if o.wg != nil {
		o.wg.Add(1)
	}

	select {
	case <-o.done:
		if o.wg != nil {
			o.wg.Done()
		}
		atomic.StoreInt32(&o.isOpen, 0)
		return false
	case o.spooler.Channel <- event:
		return true
	}
}
