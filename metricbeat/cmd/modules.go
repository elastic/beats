package cmd

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/cmd"
)

func buildModulesManager(beat *beat.Beat) (cmd.ModulesManager, error) {
	config := beat.BeatConfig

	glob, err := config.String("config.modules.path", -1)
	if err != nil {
		return nil, errors.Errorf("modules management requires 'metricbeat.config.modules.path' setting")
	}

	if !strings.HasSuffix(glob, "*.yml") {
		return nil, errors.Errorf("wrong settings for config.modules.path, it is expected to end with *.yml. Got: %s", glob)
	}

	modulesManager, err := cfgfile.NewGlobManager(glob, ".yml", ".disabled")
	if err != nil {
		return nil, errors.Wrap(err, "initialization error")
	}
	return modulesManager, nil
}
