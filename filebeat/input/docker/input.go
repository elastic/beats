package docker

import (
	"path"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/filebeat/input/log"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"

	"github.com/pkg/errors"
)

func init() {
	err := input.Register("docker", NewInput)
	if err != nil {
		panic(err)
	}
}

// NewInput creates a new docker input
func NewInput(cfg *common.Config, outletFactory channel.Factory, context input.Context) (input.Input, error) {
	cfgwarn.Experimental("Docker input is enabled.")

	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrap(err, "reading docker input config")
	}

	// Wrap log input with custom docker settings
	if len(config.Containers.IDs) > 0 {
		for idx, containerID := range config.Containers.IDs {
			cfg.SetString("paths", idx, path.Join(config.Containers.Path, containerID, "*.log"))
		}
	}

	if err := cfg.SetBool("docker-json", -1, true); err != nil {
		return nil, errors.Wrap(err, "update input config")
	}
	return log.NewInput(cfg, outletFactory, context)
}
