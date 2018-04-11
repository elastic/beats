package fileset

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestProspectorDeprecation(t *testing.T) {
	cfg := map[string]interface{}{
		"enabled": true,
		"prospector": map[string]interface{}{
			"close_eof": true,
		},
	}

	c, err := common.NewConfigFrom(cfg)
	assert.NoError(t, err)

	f, err := NewFilesetConfig(c)
	if assert.NoError(t, err) {
		assert.Equal(t, f.Input["close_eof"], true)
	}
}

func TestInputSettings(t *testing.T) {
	cfg := map[string]interface{}{
		"enabled": true,
		"input": map[string]interface{}{
			"close_eof": true,
		},
	}

	c, err := common.NewConfigFrom(cfg)
	assert.NoError(t, err)

	f, err := NewFilesetConfig(c)
	if assert.NoError(t, err) {
		assert.Equal(t, f.Input["close_eof"], true)
		assert.Nil(t, f.Prospector)
	}
}

func TestProspectorDeprecationWhenInputIsAlsoDefined(t *testing.T) {
	cfg := map[string]interface{}{
		"enabled": true,
		"input": map[string]interface{}{
			"close_eof": true,
		},
		"prospector": map[string]interface{}{
			"close_eof": true,
		},
	}

	c, err := common.NewConfigFrom(cfg)
	assert.NoError(t, err)

	_, err = NewFilesetConfig(c)
	assert.Error(t, err, "error prospector and input are defined in the fileset, use only input")
}
