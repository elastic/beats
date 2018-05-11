package scheduling

import "github.com/elastic/beats/libbeat/common"

// Config defines global scheduling configurations.
type Config struct {
	Groups   map[string][]common.ConfigNamespace `config:"groups"`
	Policies []common.ConfigNamespace            `config:"policies"`
}

type LocalConfig struct {
	Group    string                   `config:"group"`
	Policies []common.ConfigNamespace `config:"policies"`
}
