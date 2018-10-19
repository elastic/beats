package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestCodeKey(t *testing.T) {
	t.Run("same bytes content return the same key", func(t *testing.T) {
		name := "hello"
		content, err := common.RandomBytes(100)
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, codeKey(name, content), codeKey(name, content))
	})

	t.Run("different bytes return a different key", func(t *testing.T) {
		name := "hello"
		content, err := common.RandomBytes(100)
		if !assert.NoError(t, err) {
			return
		}

		other, err := common.RandomBytes(100)
		if !assert.NoError(t, err) {
			return
		}

		assert.NotEqual(t, codeKey(name, content), codeKey(name, other))
	})
}
