package harvester

import (
	"testing"
	"time"

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

func TestCloseOlder(t *testing.T) {

	config := defaultConfig
	assert.Equal(t, config.CloseOlder, 0*time.Hour)
	assert.Equal(t, config.CloseInactive, defaultConfig.CloseInactive)

	config.CloseOlder = 5 * time.Hour
	config.Validate()

	assert.Equal(t, config.CloseInactive, 5*time.Hour)
}
