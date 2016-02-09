package redis

import (
	"testing"

	//"github.com/elastic/beats/libbeat/logp"
	"github.com/stretchr/testify/assert"
)

func TestConnect(t *testing.T) {

	if testing.Short() {
		t.Skip("Skipping in short mode, because it requires Redis")
	}

	_, err := Connect(GetRedisEnvHost() + ":" + GetRedisEnvPort())
	assert.NoError(t, err)
}
