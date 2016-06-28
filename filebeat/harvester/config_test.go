package harvester

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestForceCloseFiles(t *testing.T) {

	config := defaultConfig
	assert.False(t, config.ForceCloseFiles)
	assert.False(t, config.CloseRemoved)
	assert.False(t, config.CloseRenamed)

	config.ForceCloseFiles = true
	config.Validate()

	assert.True(t, config.ForceCloseFiles)
	assert.True(t, config.CloseRemoved)
	assert.True(t, config.CloseRenamed)
}
