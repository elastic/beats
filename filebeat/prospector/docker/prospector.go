package docker

import (
	"path"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/prospector"
	"github.com/elastic/beats/filebeat/prospector/log"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"

	"github.com/pkg/errors"
)

func init() {
	err := prospector.Register("docker", NewProspector)
	if err != nil {
		panic(err)
	}
}

// NewProspector creates a new docker prospector
func NewProspector(cfg *common.Config, outletFactory channel.Factory, context prospector.Context) (prospector.Prospectorer, error) {
	cfgwarn.Experimental("Docker prospector is enabled.")

	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrap(err, "reading docker prospector config")
	}

	// Wrap log prospector with custom docker settings
	if len(config.Containers.IDs) > 0 {
		for idx, containerID := range config.Containers.IDs {
			cfg.SetString("paths", idx, path.Join(config.Containers.Path, containerID, "*.log"))
		}
	}

	if err := cfg.SetBool("docker-json", -1, true); err != nil {
		return nil, errors.Wrap(err, "update prospector config")
	}
	return log.NewProspector(cfg, outletFactory, context)
}
