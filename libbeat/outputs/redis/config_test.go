package redis

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	type io struct {
		Name  string
		Input redisConfig
		Valid bool
	}

	tests := []io{
		io{"No config", redisConfig{Key: "", Index: ""}, true},
		io{"Only key", redisConfig{Key: "test", Index: ""}, true},
		io{"Only index", redisConfig{Key: "", Index: "test"}, true},
		io{"Both", redisConfig{Key: "test", Index: "test"}, false},

		io{"Invalid Datatype", redisConfig{Key: "test", DataType: "something"}, false},
		io{"List Datatype", redisConfig{Key: "test", DataType: "list"}, true},
		io{"Channel Datatype", redisConfig{Key: "test", DataType: "channel"}, true},
	}

	for _, test := range tests {
		assert.Equal(t, test.Input.Validate() == nil, test.Valid, test.Name)
	}
}
