package consumergroup

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/outputs"
)

type metricsetConfig struct {
	Retries  int                `config:"retries" validate:"min=0"`
	Backoff  time.Duration      `config:"backoff" validate:"min=0"`
	TLS      *outputs.TLSConfig `config:"ssl"`
	Username string             `config:"username"`
	Password string             `config:"password"`
	ClientID string             `config:"client_id"`

	Groups []string `config:"groups"`
	Topics []string `config:"topics"`
}

var defaultConfig = metricsetConfig{
	Retries:  3,
	Backoff:  250 * time.Millisecond,
	TLS:      nil,
	Username: "",
	Password: "",
	ClientID: "metricbeat",
}

func (c *metricsetConfig) Validate() error {
	if c.Username != "" && c.Password == "" {
		return fmt.Errorf("password must be set when username is configured")
	}

	return nil
}
