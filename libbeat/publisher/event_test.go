package publisher

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOutputListener_NoNilCheckRequired(t *testing.T) {
	o := OutputListener{}

	assert.NotPanics(t,
		func() {
			o.NewEvent()
			o.Acked()
			o.Dropped()
			o.DeadLetter()
		},
		"Calling methods on a zero value OutputListener must not panic")
}
