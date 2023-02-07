package features

import (
	"fmt"
	"sync"

	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

var (
	// flags configs
	mu sync.Mutex

	flags fflags
)

type fflags struct {
	fqdnEnabled bool
}

// type configs struct {
// 	FQDN struct {
// 		Enabled bool `json:"enabled" yaml:"enabled" config:"enabled"`
// 	} `json:"fqdn" yaml:"fqdn" config:"fqdn"`
// }

func Update(f *proto.Features) {
	if f == nil {
		return
	}

	mu.Lock()
	defer mu.Unlock()
	flags = fflags{fqdnEnabled: f.Fqdn.Enabled}
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
	flags.fqdnEnabled = enabled

	return nil
}

// FQDN reports if FQDN should be used instead of hostname for host.name.
func FQDN() bool {
	return flags.fqdnEnabled
}
