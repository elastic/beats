// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import "github.com/elastic/beats/libbeat/common"

type Config struct {
	// Modules is a list of module specific configuration data.
	Monitors  []*common.Config `config:"monitors"         validate:"required"`
	Scheduler Scheduler        `config:"scheduler"`
}

type Scheduler struct {
	Limit    uint   `config:"limit"  validate:"min=0"`
	Location string `config:"location"`
}

var DefaultConfig = Config{}
