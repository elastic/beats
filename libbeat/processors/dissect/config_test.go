package dissect

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestTokenizerType(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		c, err := common.NewConfigFrom(map[string]interface{}{
			"tokenizer": "%{value1}",
			"field":     "message",
		})
		if !assert.NoError(t, err) {
			return
		}

		cfg := config{}
		err = c.Unpack(&cfg)
		if !assert.NoError(t, err) {
			return
		}
	})

	t.Run("invalid", func(t *testing.T) {
		c, err := common.NewConfigFrom(map[string]interface{}{
			"tokenizer": "%value1}",
			"field":     "message",
		})
		if !assert.NoError(t, err) {
			return
		}

		cfg := config{}
		err = c.Unpack(&cfg)
		if !assert.Error(t, err) {
			return
		}
	})
}
