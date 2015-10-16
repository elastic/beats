package beat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewBeat(t *testing.T) {

	tb := &TestBeater{}
	b := NewBeat("testbeat", "0.9", tb)

	assert.Equal(t, "testbeat", b.Name)
	assert.Equal(t, "0.9", b.Version)
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
