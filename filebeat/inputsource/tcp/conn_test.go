package tcp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResetableLimitedReader(t *testing.T) {
	maxReadBuffer := 400

	t.Run("WhenMaxReadIsReachedInMultipleRead", func(t *testing.T) {
		r := strings.NewReader(randomString(maxReadBuffer * 2))
		m := NewResetableLimitedReader(r, uint64(maxReadBuffer))
		toRead := make([]byte, maxReadBuffer)
		_, err := m.Read(toRead)
		assert.NoError(t, err)
		toRead = make([]byte, 300)
		_, err = m.Read(toRead)
		assert.Equal(t, ErrMaxReadBuffer, err)
	})

	t.Run("WhenMaxReadIsNotReached", func(t *testing.T) {
		r := strings.NewReader(randomString(maxReadBuffer * 2))
		m := NewResetableLimitedReader(r, uint64(maxReadBuffer))
		toRead := make([]byte, maxReadBuffer)
		_, err := m.Read(toRead)
		assert.NoError(t, err)
	})

	t.Run("WhenResetIsCalled", func(t *testing.T) {
		r := strings.NewReader(randomString(maxReadBuffer * 2))
		m := NewResetableLimitedReader(r, uint64(maxReadBuffer))
		toRead := make([]byte, maxReadBuffer)
		_, err := m.Read(toRead)
		assert.NoError(t, err)
		m.Reset()
		toRead = make([]byte, 300)
		_, err = m.Read(toRead)
		assert.NoError(t, err)
	})
}
