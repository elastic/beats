package kibana

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientConfigValdiate(t *testing.T) {
	tests := []struct {
		name string
		c    *ClientConfig
		err  error
	}{{
		name: "empty params",
		c:    &ClientConfig{},
		err:  nil,
	}, {
		name: "username and password",
		c: &ClientConfig{
			Username: "user",
			Password: "pass",
		},
		err: nil,
	}, {
		name: "api_key",
		c: &ClientConfig{
			APIKey: "api-key",
		},
		err: nil,
	}, {
		name: "username and api_key",
		c: &ClientConfig{
			Username: "user",
			APIKey:   "apiKey",
		},
		err: fmt.Errorf("cannot set both api_key and username/password"),
	}}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := tt.c.Validate()
			if tt.err == nil {
				assert.Nil(t, err)
			} else {
				assert.EqualError(t, err, tt.err.Error())
			}
		})
	}

}
