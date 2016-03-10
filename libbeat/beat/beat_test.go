// +build !integration

package beat

import (
	"testing"

	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func Test_NewBeat(t *testing.T) {

	tb := &TestBeater{}
	b := NewBeat("testbeat", "0.9", tb)

	assert.Equal(t, "testbeat", b.Name)
	assert.Equal(t, "0.9", b.Version)

	// UUID4 should be 36 chars long
	assert.Equal(t, 16, len(b.UUID))
	assert.Equal(t, 36, len(b.UUID.String()))
}

func Test_NewBeat_UUID(t *testing.T) {

	tb := &TestBeater{}
	b := NewBeat("testbeat", "0.9", tb)

	// Make sure the UUID's are different
	assert.NotEqual(t, b.UUID, uuid.NewV4())
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
