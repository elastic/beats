package registries

import (
	"strings"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
)

// prefix registry routes configuration with a common prefix to the given registry.
// If a configured input type does not has the prefix of the prefixRegistry,
// the registry fails.
type prefixRegistry struct {
	v2.Registry
	prefix string
}

// Prefixed wraps a Registry into a prefixRegistry. All inputs in the input registry are now addressable via the common prefix only.
// For example this setup:
//
//    reg = withTypePrefix("logs", filebeatInputs)
//
// requires the input configuration to load the journald input like this:
//
//    - type: logs/journald
//
func Prefixed(name string, reg v2.Registry) v2.Registry {
	return &prefixRegistry{Registry: reg, prefix: name + "/"}
}

func (r *prefixRegistry) Find(name string) (v2.Plugin, bool) {
	if !strings.HasPrefix(name, r.prefix) {
		return v2.Plugin{}, false
	}
	return r.Registry.Find(name[len(r.prefix):])
}
