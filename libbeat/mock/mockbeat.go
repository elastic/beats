package mock

import (
	"github.com/elastic/beats/libbeat/beat"
)

///*** Mock Beat Setup ***///

var Version = "9.9.9"
var Name = "mockbeat"

type Mockbeat struct {
	done chan struct{}
}

// Creates beater
func New() *Mockbeat {
	return &Mockbeat{
		done: make(chan struct{}),
	}
}

/// *** Beater interface methods ***///

func (mb *Mockbeat) Config(b *beat.Beat) error {
	return nil
}

func (mb *Mockbeat) Setup(b *beat.Beat) error {
	return nil
}

func (mb *Mockbeat) Run(b *beat.Beat) error {
	// Wait until mockbeat is done
	<-mb.done
	return nil
}

func (mb *Mockbeat) Cleanup(b *beat.Beat) error {
	return nil
}

func (mb *Mockbeat) Stop() {
	close(mb.done)
}
