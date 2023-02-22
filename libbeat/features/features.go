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

// UpdateFromProto updates the feature flags config. If f is nil UpdateFromProto is no-op.
func UpdateFromProto(f *proto.Features) {
	logp.L().Info("[features] features.UpdateFromProto fqdn invoked")
	if f == nil {
		logp.L().Infof("[features] features.UpdateFromProto received nil: %v", f)
		return
	}

	if f.Fqdn == nil {
		f.Fqdn = &proto.FQDNFeature{}
	}

	mu.Lock()
	defer mu.Unlock()
	flags = fflags{fqdnEnabled: f.Fqdn.Enabled}

	logp.L().Infof("[features] features.UpdateFromProto: fqdn: %t", flags.fqdnEnabled)
}

func UpdateFromConfig(c *conf.C) error {
	logp.L().Info("[features] features.UpdateFromConfig invoked")
	if c == nil {
		logp.L().Info("[features] feature flag config is nil!")
		return nil
	}

	type cfg struct {
		Features struct {
			FQDN *conf.C `json:"fqdn" yaml:"fqdn" config:"fqdn"`
		} `json:"features" yaml:"features" config:"features"`
	}

	parsedFlags := cfg{}
	if err := c.Unpack(&parsedFlags); err != nil {
		logp.L().Errorf("[features] could not Unpack features config: %v", err)
		return fmt.Errorf("could not Unpack features config: %w", err)
	}

	mu.Lock()
	defer mu.Unlock()
	flags = fflags{fqdnEnabled: parsedFlags.Features.FQDN.Enabled()}

	logp.L().Infof("[features] features.UpdateFromConfig: fqdn: %t", flags.fqdnEnabled)

	return nil
}

// FQDN reports if FQDN should be used instead of hostname for host.name.
// If it hasn't been set by UpdateFromConfig or UpdateFromProto, it returns false.
func FQDN() bool {
	mu.Lock()
	defer mu.Unlock()
	return flags.fqdnEnabled
}
