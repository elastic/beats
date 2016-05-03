// +build integration

package redis

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnect(t *testing.T) {
	_, err := Connect(GetRedisEnvHost() + ":" + GetRedisEnvPort())
	assert.NoError(t, err)
}
