package features

import (
	"fmt"
	"sync"

	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
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

func UpdateFromProto(f *proto.Features) {
	logp.L().Info("features.UpdateFromProto invoked")

	if f == nil {
		logp.L().Info("feature flag proto is nil!")
		return
	}

	mu.Lock()
	defer mu.Unlock()
	flags = fflags{fqdnEnabled: f.Fqdn.Enabled}

	logp.L().Infof("features.UpdateFromProto: fqdn: %t", flags.fqdnEnabled)
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
// If it hasn't been set by ParseFromConfig or UpdateFromProto, it returns false.
func FQDN() bool {
	return flags.fqdnEnabled
}
