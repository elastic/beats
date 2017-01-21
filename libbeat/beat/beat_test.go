// +build !integration

package beat

import (
	"testing"

	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewInstance(t *testing.T) {
	b := newBeat("testbeat", "0.9")

	assert.Equal(t, "testbeat", b.Name)
	assert.Equal(t, "0.9", b.Version)

	// UUID4 should be 36 chars long
	assert.Equal(t, 16, len(b.UUID))
	assert.Equal(t, 36, len(b.UUID.String()))
}

func TestNewInstanceUUID(t *testing.T) {
	b := newBeat("testbeat", "0.9")

	// Make sure the UUID's are different
	assert.NotEqual(t, b.UUID, uuid.NewV4())
}
