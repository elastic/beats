package mongodbatlas

import (
	"fmt"
	"time"
)

type Config struct {
	GroupId    stringList `config:"group_id,replace"`
	LogName    stringList `config:"log_name,replace" validate:"required"`
	PublicKey  string     `config:"public_key" validate:"required"`
	PrivateKey string     `config:"private_key" validate:"required"`

	// API contains settings to adapt to changes on the API.
	API APIConfig `config:"api"`
}

type APIConfig struct {
	// ErrorRetryInterval sets the interval between retries in the case of
	// errors performing a request.
	ErrorRetryInterval time.Duration `config:"error_retry_interval" validate:"positive"`

	// MaxRequestsPerMinute sets the limit on the number of API requests that
	// can be sent, per tenant.
	MaxRequestsPerMinute int `config:"max_requests_per_minute" validate:"positive"`

	// MaxRetention determines how far back the input will poll for events.
	MaxRetention time.Duration `config:"max_retention" validate:"positive"`
}

func defaultConfig() Config {
	return Config{
		GroupId: []string{},
		LogName: []string{
			"mongodb.gz",
			"mongos.gz",
		},
		API: APIConfig{
			ErrorRetryInterval:   5 * time.Minute,
			MaxRequestsPerMinute: 100,
			MaxRetention:         1 * time.Minute,
		},
	}
}

type stringList []string

// Unpack populates the stringList with either a single string value or an array.
func (s *stringList) Unpack(value interface{}) error {
	switch v := value.(type) {
	case string:
		*s = []string{v}
	case []string:
		*s = v
	case []interface{}:
		*s = make([]string, len(v))
		for idx, ival := range v {
			str, ok := ival.(string)
			if !ok {
				return fmt.Errorf("string value required. Found %v (type %T) at position %d",
					ival, ival, idx+1)
			}
			(*s)[idx] = str
		}
	default:
		return fmt.Errorf("array of strings required. Found %v (type %T)", value, value)
	}
	return nil
}
