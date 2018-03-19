// +build !integration

package instance

import (
	"testing"

	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewInstance(t *testing.T) {
	b, err := NewBeat("testbeat", "testidx", "0.9")
	if err != nil {
		panic(err)
	}

	assert.Equal(t, "testbeat", b.Info.Beat)
	assert.Equal(t, "testidx", b.Info.IndexPrefix)
	assert.Equal(t, "0.9", b.Info.Version)

	// UUID4 should be 36 chars long
	assert.Equal(t, 16, len(b.Info.UUID))
	assert.Equal(t, 36, len(b.Info.UUID.String()))

	// indexPrefix set to name if empty
	b, err = NewBeat("testbeat", "", "0.9")
	if err != nil {
		panic(err)
	}
	assert.Equal(t, "testbeat", b.Info.Beat)
	assert.Equal(t, "testbeat", b.Info.IndexPrefix)

}

func TestNewInstanceUUID(t *testing.T) {
	b, err := NewBeat("testbeat", "", "0.9")
	if err != nil {
		panic(err)
	}

	// Make sure the UUID's are different
	assert.NotEqual(t, b.Info.UUID, uuid.NewV4())
}
