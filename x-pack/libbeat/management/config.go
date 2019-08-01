// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"io"
	"text/template"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/libbeat/kibana"
)

// ManagedConfigTemplate is used to overwrite settings file during enrollment
const ManagedConfigTemplate = `

#========================= Central Management =================================

# Beats is configured under central management, you can define most settings
# from the Kibana UI. You can update this file to configure the settings that
# are not supported by Kibana Beats management.

{{.CentralManagementSettings}}
#================================ General =====================================

# The name of the shipper that publishes the network data. It can be used to group
# all the transactions sent by a single shipper in the web interface.
#name:

# The tags of the shipper are included in their own field with each
# transaction published.
#tags: ["service-X", "web-tier"]

# Optional fields that you can specify to add additional information to the
# output.
#fields:
#  env: staging

#================================ Logging =====================================

# Sets log level. The default log level is info.
# Available log levels are: error, warning, info, debug
#logging.level: debug

# At debug level, you can selectively enable logging only for some components.
# To enable all selectors use ["*"]. Examples of other selectors are "beat",
# "publish", "service".
#logging.selectors: ["*"]

#============================== Xpack Monitoring ===============================
# {{.BeatName}} can export internal metrics to a central Elasticsearch monitoring
# cluster.  This requires xpack monitoring to be enabled in Elasticsearch.  The
# reporting is disabled by default.

# Set to true to enable the monitoring reporter.
#monitoring.enabled: false

# Uncomment to send the metrics to Elasticsearch. Most settings from the
# Elasticsearch output are accepted here as well.
# Note that the settings should point to your Elasticsearch *monitoring* cluster.
# Any setting that is not set is automatically inherited from the Elasticsearch
# output configuration, so if you have the Elasticsearch output configured such
# that it is pointing to your Elasticsearch monitoring cluster, you can simply
# uncomment the following line.
#monitoring.elasticsearch:
`

const (
	// ModeCentralManagement is a default CM mode, using existing processes
	ModeCentralManagement = "cm"

	// ModeFleet is a management mode where fleet is used to retrieve configurations
	ModeFleet = "fleet"
)

// Config for central management
type Config struct {
	// true when enrolled
	Enabled bool `config:"enabled" yaml:"enabled"`

	// Mode specifies whether beat uses Central Management or Fleet.
	// Options: [cm, fleet]
	Mode string `config:"mode" yaml:"mode"`

	// Poll configs period
	Period time.Duration `config:"period" yaml:"period"`

	EventsReporter EventReporterConfig `config:"events_reporter" yaml:"events_reporter"`

	AccessToken string `config:"access_token" yaml:"access_token"`

	Kibana *kibana.ClientConfig `config:"kibana" yaml:"kibana"`

	Blacklist ConfigBlacklistSettings `config:"blacklist" yaml:"blacklist"`
}

// EventReporterConfig configuration of the events reporter.
type EventReporterConfig struct {
	Period       time.Duration `config:"period" yaml:"period"`
	MaxBatchSize int           `config:"max_batch_size" yaml:"max_batch_size" validate:"nonzero,positive"`
}

func defaultConfig() *Config {
	return &Config{
		Mode:   ModeCentralManagement,
		Period: 60 * time.Second,
		EventsReporter: EventReporterConfig{
			Period:       30 * time.Second,
			MaxBatchSize: 1000,
		},
		Blacklist: ConfigBlacklistSettings{
			Patterns: map[string]string{
				"output": "console|file",
			},
		},
	}
}

type templateParams struct {
	CentralManagementSettings string
	BeatName                  string
}

// OverwriteConfigFile will overwrite beat settings file with the enrolled template
func (c *Config) OverwriteConfigFile(wr io.Writer, beatName string) error {
	t := template.Must(template.New("beat.management.yml").Parse(ManagedConfigTemplate))

	tmp := struct {
		Management *Config `yaml:"management"`
	}{
		Management: c,
	}

	data, err := yaml.Marshal(tmp)
	if err != nil {
		return err
	}

	params := templateParams{
		CentralManagementSettings: string(data),
		BeatName:                  beatName,
	}

	return t.Execute(wr, params)
}
