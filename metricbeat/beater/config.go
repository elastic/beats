package beater

import "github.com/elastic/beats/libbeat/common"

// Config is the root of the Metricbeat configuration hierarchy.
type Config struct {
	// Modules is a list of module specific configuration data.
	Modules []*common.Config `config:"modules" validate:"required"`
}
