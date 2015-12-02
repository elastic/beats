package mock

import (
	"fmt"

	"github.com/elastic/libbeat/beat"
)

///*** Mock Beat Setup ***///

var Version = "0.0.1"
var Name = "mockbeat"

type Mockbeat struct {
}

func (mb *Mockbeat) Config(b *beat.Beat) error {
	fmt.Print("hello world")
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
