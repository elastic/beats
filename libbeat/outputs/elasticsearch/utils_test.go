package elasticsearch

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestTryLowercaseIndex(t *testing.T) {
	testCases := map[string]string{
		"ElasticSearch": "elasticsearch",
		"ELASTIC":       "elastic",
		"beats":         "beats",
	}

	for former, latter := range testCases {
		assert.Equal(t, latter, TryLowercaseIndex(former))
	}
}
