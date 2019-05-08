package channel

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/filebeat/util"
	"github.com/elastic/beats/libbeat/tests/resources"
)

type dummyOutletter struct {
	closed bool
	c      chan struct{}
}

func (o *dummyOutletter) OnEvent(event *util.Data) bool {
	return true
}

func (o *dummyOutletter) Close() error {
	o.closed = true
	close(o.c)
	return nil
}

func (o *dummyOutletter) Done() <-chan struct{} {
	return o.c
}

func TestCloseOnSignal(t *testing.T) {
	resources.CheckGoroutines(t, func() {
		o := &dummyOutletter{c: make(chan struct{})}
		sig := make(chan struct{})
		CloseOnSignal(o, sig)
		close(sig)
	})
}

func TestCloseOnSignalClosed(t *testing.T) {
	resources.CheckGoroutines(t, func() {
		o := &dummyOutletter{c: make(chan struct{})}
		sig := make(chan struct{})
		c := CloseOnSignal(o, sig)
		c.Close()
	})
}

func TestSubOutlet(t *testing.T) {
	resources.CheckGoroutines(t, func() {
		o := &dummyOutletter{c: make(chan struct{})}
		so := SubOutlet(o)
		so.Close()
		assert.False(t, o.closed)
	})
}

func TestCloseOnSignalSubOutlet(t *testing.T) {
	resources.CheckGoroutines(t, func() {
		o := &dummyOutletter{c: make(chan struct{})}
		c := CloseOnSignal(SubOutlet(o), make(chan struct{}))
		o.Close()
		c.Close()
	})
}
