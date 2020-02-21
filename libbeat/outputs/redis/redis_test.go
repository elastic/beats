package redis

import (
	"testing"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs"
	_ "github.com/elastic/beats/libbeat/outputs/codec/json"
	"github.com/stretchr/testify/assert"
)

func TestMakeRedis(t *testing.T) {
	tests := []struct {
		Name                 string
		Config               map[string]interface{}
		Valid                bool
		AdditionalValidation func(*testing.T, outputs.Group)
	}{
		{
			"No host",
			map[string]interface{}{
				"hosts": []string{},
			}, false,
			nil,
		},
		{
			"Invalid scheme",
			map[string]interface{}{
				"hosts": []string{"redisss://localhost:6379"},
			}, false,
			nil,
		},
		{
			"Single host",
			map[string]interface{}{
				"hosts": []string{"localhost:6379"},
			}, true,
			func(t2 *testing.T, groups outputs.Group) {
				assert.Len(t2, groups.Clients, 1)
				redisClient := groups.Clients[0].(*backoffClient)
				assert.Empty(t2, redisClient.client.password)
			},
		},
		{
			"Multiple hosts",
			map[string]interface{}{
				"hosts": []string{"redis://localhost:6379", "rediss://localhost:6380"},
			}, true,
			func(t2 *testing.T, groups outputs.Group) {
				assert.Len(t2, groups.Clients, 2)
			},
		},
		{
			"Default password",
			map[string]interface{}{
				"hosts":    []string{"redis://localhost:6379"},
				"password": "defaultPassword",
			}, true,
			func(t2 *testing.T, groups outputs.Group) {
				assert.Len(t2, groups.Clients, 1)
				redisClient := groups.Clients[0].(*backoffClient)
				assert.Equal(t2, "defaultPassword", redisClient.client.password)
			},
		},
		{
			"Specific and default password",
			map[string]interface{}{
				"hosts":    []string{"redis://localhost:6379", "rediss://:mypassword@localhost:6380"},
				"password": "defaultPassword",
			}, true,
			func(t2 *testing.T, groups outputs.Group) {
				assert.Len(t2, groups.Clients, 2)
				redisClient := groups.Clients[0].(*backoffClient)
				assert.Equal(t2, "defaultPassword", redisClient.client.password)
				redisClient = groups.Clients[1].(*backoffClient)
				assert.Equal(t2, "mypassword", redisClient.client.password)
			},
		},
	}
	beatInfo := beat.Info{Beat: "libbeat", Version: "1.2.3"}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			cfg, err := common.NewConfigFrom(test.Config)
			assert.NoError(t, err)
			groups, err := makeRedis(nil, beatInfo, outputs.NewNilObserver(), cfg)
			assert.Equal(t, err == nil, test.Valid)
			if err != nil && test.Valid {
				t.Log(err)
			}
			if test.AdditionalValidation != nil {
				test.AdditionalValidation(t, groups)
			}
		})
	}
}
