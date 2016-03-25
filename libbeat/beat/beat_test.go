// +build !integration

package beat

import (
	"testing"

	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewInstance(t *testing.T) {
	tb := &TestBeater{}
	b := newInstance("testbeat", "0.9", tb)

	assert.Equal(t, "testbeat", b.data.Name)
	assert.Equal(t, "0.9", b.data.Version)

	// UUID4 should be 36 chars long
	assert.Equal(t, 16, len(b.data.UUID))
	assert.Equal(t, 36, len(b.data.UUID.String()))
}

func TestNewInstanceUUID(t *testing.T) {
	tb := &TestBeater{}
	b := newInstance("testbeat", "0.9", tb)

	// Make sure the UUID's are different
	assert.NotEqual(t, b.data.UUID, uuid.NewV4())
}

// Test beat object
type TestBeater struct {
}

func (tb *TestBeater) Config(b *Beat) error {
	return nil
}
func (tb *TestBeater) Setup(b *Beat) error {
	return nil
}
func (tb *TestBeater) Run(b *Beat) error {
	return nil
}

func (tb *TestBeater) Cleanup(b *Beat) error {
	return nil
}

func (tb *TestBeater) Stop() {
}
