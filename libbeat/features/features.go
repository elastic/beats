package features

import (
	"fmt"
	"sync"

	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

var (
	mu sync.Mutex

	flags fflags
)

type fflags struct {
	fqdnEnabled bool
}

func Update(f client.Features) {
	logp.L().Info("features.Update fqdn invoked")

	mu.Lock()
	defer mu.Unlock()
	flags = fflags{fqdnEnabled: f.FQDN.Enabled}

	logp.L().Infof("features.Update: fqdn: %t", flags.fqdnEnabled)
}

func ParseFromConfig(c *conf.C) error {
	logp.L().Info("features.ParseFromConfig invoked")
	if c == nil {
		logp.L().Info("feature flag config is nil!")
		return nil
	}

	type cfg struct {
		Features struct {
			FQDN *conf.C `json:"fqdn" yaml:"fqdn" config:"fqdn"`
		} `json:"features" yaml:"features" config:"features"`
	}

	parsedFlags := cfg{}
	if err := c.Unpack(&parsedFlags); err != nil {
		logp.L().Errorf("could not Unpack features config: %v", err)
		return fmt.Errorf("could not Unpack features config: %w", err)
	}

	mu.Lock()
	defer mu.Unlock()
	flags.fqdnEnabled = parsedFlags.Features.FQDN.Enabled()

	logp.L().Infof("features.ParseFromConfig: fqdn: %t", flags.fqdnEnabled)

	return nil
}

// FQDN reports if FQDN should be used instead of hostname for host.name.
// If it hasn't been set by ParseFromConfig or Update, it returns false.
func FQDN() bool {
	mu.Lock()
	defer mu.Unlock()
	return flags.fqdnEnabled
}
