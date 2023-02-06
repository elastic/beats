package features

import (
	"fmt"
	"sync"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

var (
	flags configs
	mu    sync.Mutex
)

type configs struct {
	FQDN struct {
		Enabled bool `json:"enabled" yaml:"enabled" config:"enabled"`
	} `json:"fqdn" yaml:"fqdn" config:"fqdn"`
}

func Parse(c *conf.C) error {
	logp.L().Info("features.Parse invoked")
	if c == nil {
		logp.L().Info("feature flag config is nil!")
		return nil
	}

	enabled, err := c.Bool("features.fqdn.enabled", -1)
	if err != nil {
		return fmt.Errorf("could not FQDN feature config: %w", err)
	}

	mu.Lock()
	defer mu.Unlock()
	flags.FQDN.Enabled = enabled

	return nil
}

// FQDN reports if FQDN should be used instead of hostname for host.name.
func FQDN() bool {
	return flags.FQDN.Enabled
}
