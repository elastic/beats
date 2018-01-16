package docker

import (
	"fmt"
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
	if len(config.Containers.IDs) == 0 {
		return nil, errors.New("Docker prospector requires at least one entry under 'containers.ids'")
	}

	for idx, containerID := range config.Containers.IDs {
		cfg.SetString("paths", idx, path.Join(config.Containers.Path, containerID, "*.log"))
	}

	if err := checkStream(config.Containers.Stream); err != nil {
		return nil, err
	}

	if err := cfg.SetString("docker-json", -1, config.Containers.Stream); err != nil {
		return nil, errors.Wrap(err, "update prospector config")
	}
	return log.NewProspector(cfg, outletFactory, context)
}

func checkStream(val string) error {
	for _, s := range []string{"all", "stdout", "stderr"} {
		if s == val {
			return nil
		}
	}

	return fmt.Errorf("Invalid value for containers.stream: %s, supported values are: all, stdout, stderr", val)
}
