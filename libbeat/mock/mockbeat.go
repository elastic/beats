package mock

import (
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/logp"
)

///*** Mock Beat Setup ***///

var Version = "9.9.9"
var Name = "mockbeat"

type Mockbeat struct {
	done chan bool
}

func (mb *Mockbeat) Config(b *beat.Beat) error {
	logp.Info("MockBeat: Config")
	return nil
}

func (mb *Mockbeat) Setup(b *beat.Beat) error {
	logp.Info("MockBeat: Setup")
	mb.done = make(chan bool)

	return nil
}

func (mb *Mockbeat) Run(b *beat.Beat) error {
	logp.Info("MockBeat: Run")

	defer func() {
		logp.Info("MockBeat: returning from Run function")
	}()

	logp.Info("MockBeat: waiting to be done")
	<-mb.done

	return nil
}

func (mb *Mockbeat) Cleanup(b *beat.Beat) error {
	logp.Info("MockBeat: Cleanup")
	return nil
}

func (mb *Mockbeat) Stop() {
	logp.Info("MockBeat: Stop")
	close(mb.done)
}
