package mock

import (
	"github.com/elastic/beats/libbeat/beat"
)

///*** Mock Beat Setup ***///

var Version = "9.9.9"
var Name = "mockbeat"

type Mockbeat struct {
}

func (mb *Mockbeat) Config(b *beat.Beat) error {
	return nil
}

func (mb *Mockbeat) Setup(b *beat.Beat) error {
	return nil
}

func (mb *Mockbeat) Run(b *beat.Beat) error {
	return nil
}

func (mb *Mockbeat) Cleanup(b *beat.Beat) error {
	return nil
}

func (mb *Mockbeat) Stop() {

}
