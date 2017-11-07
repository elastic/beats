package redis

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		Name  string
		Input redisConfig
		Valid bool
	}{
		{"No config", redisConfig{Key: "", Index: ""}, true},
		{"Only key", redisConfig{Key: "test", Index: ""}, true},
		{"Only index", redisConfig{Key: "", Index: "test"}, true},
		{"Both", redisConfig{Key: "test", Index: "test"}, false},

		{"Invalid Datatype", redisConfig{Key: "test", DataType: "something"}, false},
		{"List Datatype", redisConfig{Key: "test", DataType: "list"}, true},
		{"Channel Datatype", redisConfig{Key: "test", DataType: "channel"}, true},
	}

	for _, test := range tests {
		assert.Equal(t, test.Input.Validate() == nil, test.Valid, test.Name)
	}
}
