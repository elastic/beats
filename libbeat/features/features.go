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

// UpdateFromProto updates the feature flags configuration. If f is nil UpdateFromProto is no-op.
func UpdateFromProto(f *proto.Features) {
	if f == nil {
		logp.L().Debug("feature flags are nil, ignoring them")
		return
	}

	if f.Fqdn == nil {
		f.Fqdn = &proto.FQDNFeature{}
	}

	mu.Lock()
	defer mu.Unlock()
	flags = fflags{fqdnEnabled: f.Fqdn.Enabled}
}

// UpdateFromConfig updates the feature flags configuration. If c is nil UpdateFromProto is no-op.
func UpdateFromConfig(c *conf.C) error {
	if c == nil {
		logp.L().Debug("feature flags are nil, ignoring them")
		return nil
	}

	type cfg struct {
		Features struct {
			FQDN *conf.C `json:"fqdn" yaml:"fqdn" config:"fqdn"`
		} `json:"features" yaml:"features" config:"features"`
	}

	parsedFlags := cfg{}
	if err := c.Unpack(&parsedFlags); err != nil {
		return fmt.Errorf("could not Unpack features config: %w", err)
	}

	mu.Lock()
	defer mu.Unlock()
	flags = fflags{fqdnEnabled: parsedFlags.Features.FQDN.Enabled()}

	return nil
}

// FQDN reports if FQDN should be used instead of hostname for host.name.
// If it hasn't been set by UpdateFromConfig or UpdateFromProto, it returns false.
func FQDN() bool {
	mu.Lock()
	defer mu.Unlock()
	return flags.fqdnEnabled
}
