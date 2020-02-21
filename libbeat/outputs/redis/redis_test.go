package redis

import (
	"testing"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs"
	_ "github.com/elastic/beats/libbeat/outputs/codec/json"
	"github.com/stretchr/testify/assert"
)

type checker func(*testing.T, outputs.Group)

func checks(cs ...checker) checker {
	return func(t *testing.T, g outputs.Group) {
		for _, c := range cs {
			c(t, g)
		}
	}
}

func clientsLen(required int) checker {
	return func(t *testing.T, group outputs.Group) {
		assert.Len(t, group.Clients, required)
	}
}

func clientPassword(index int, pass string) checker {
	return func(t *testing.T, group outputs.Group) {
		redisClient := group.Clients[index].(*backoffClient)
		assert.Equal(t, redisClient.client.password, pass)
	}
}

func TestMakeRedis(t *testing.T) {
	tests := map[string]struct {
		config map[string]interface{}
		valid  bool
		checks checker
	}{
		"no host": {
			config: map[string]interface{}{
				"hosts": []string{},
			},
		},
		"invald scheme": {
			config: map[string]interface{}{
				"hosts": []string{"redisss://localhost:6379"},
			},
		},
		"Single host": {
			config: map[string]interface{}{
				"hosts": []string{"localhost:6379"},
			},
			valid:  true,
			checks: checks(clientsLen(1), clientPassword(0, "")),
		},
		"Multiple hosts": {
			config: map[string]interface{}{
				"hosts": []string{"redis://localhost:6379", "rediss://localhost:6380"},
			},
			valid:  true,
			checks: clientsLen(2),
		},
		"Default password": {
			config: map[string]interface{}{
				"hosts":    []string{"redis://localhost:6379"},
				"password": "defaultPassword",
			},
			valid:  true,
			checks: checks(clientsLen(1), clientPassword(0, "defaultPassword")),
		},
		"Specific and default password": {
			config: map[string]interface{}{
				"hosts":    []string{"redis://localhost:6379", "rediss://:mypassword@localhost:6380"},
				"password": "defaultPassword",
			},
			valid: true,
			checks: checks(
				clientsLen(2),
				clientPassword(0, "defaultPassword"),
				clientPassword(1, "mypassword"),
			),
		},
	}
	beatInfo := beat.Info{Beat: "libbeat", Version: "1.2.3"}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cfg, err := common.NewConfigFrom(test.config)
			assert.NoError(t, err)
			groups, err := makeRedis(nil, beatInfo, outputs.NewNilObserver(), cfg)
			assert.Equal(t, err == nil, test.valid)
			if err != nil && test.valid {
				t.Log(err)
			}
			if test.checks != nil {
				test.checks(t, groups)
			}
		})
	}
}
