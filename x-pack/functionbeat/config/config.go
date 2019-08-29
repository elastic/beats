// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import (
	"fmt"
	"regexp"

	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
)

var (
	functionPattern = "^[A-Za-z][A-Za-z0-9\\-]{0,139}$"
	functionRE      = regexp.MustCompile(functionPattern)
	configOverrides = common.MustNewConfigFrom(map[string]interface{}{
		"path.data":              "/tmp",
		"path.logs":              "/tmp/logs",
		"keystore.path":          "/tmp/beats.keystore",
		"setup.template.enabled": true,
		"queue.mem": map[string]interface{}{
			"flush.min_events": 10,
			"flush.timeout":    "0.01s",
		},
	})
	functionLoggingOverrides = common.MustNewConfigFrom(map[string]interface{}{
		"logging.to_stderr": true,
		"logging.to_files":  false,
	})
	esOverrides = common.MustNewConfigFrom(map[string]interface{}{
		"queue.mem.events":                   "${output.elasticsearch.bulk_max_size}",
		"output.elasticsearch.bulk_max_size": 50,
	})
	logstashOverrides = common.MustNewConfigFrom(map[string]interface{}{
		"queue.mem.events": "${output.logstash.bulk_max_size}",
		"output.logstash": map[string]interface{}{
			"bulk_max_size": 50,
			"pipelining":    0,
		},
	})

	// Overrides overrides the default configuration provided by libbeat.
	Overrides = []cfgfile.ConditionalOverride{
		cfgfile.ConditionalOverride{
			Check:  always,
			Config: configOverrides,
		},
		cfgfile.ConditionalOverride{
			Check:  isLogstash,
			Config: logstashOverrides,
		},
		cfgfile.ConditionalOverride{
			Check:  isElasticsearch,
			Config: esOverrides,
		},
	}

	functionOverride = cfgfile.ConditionalOverride{
		Check:  always,
		Config: functionLoggingOverrides,
	}

	// FunctionOverrides contain logging settings
	FunctionOverrides = append(Overrides, functionOverride)
)

// Config default configuration for Functionbeat.
type Config struct {
	Provider *common.Config `config:"provider" validate:"required"`
}

// ProviderConfig is a generic configured used by providers.
type ProviderConfig struct {
	Functions []*common.Config `config:"functions"`
}

// FunctionConfig minimal configuration from each function.
type FunctionConfig struct {
	Type    string       `config:"type"`
	Name    functionName `config:"name"`
	Enabled bool         `config:"enabled"`
}

// DefaultConfig is the default configuration for Functionbeat.
var DefaultConfig = Config{}

// DefaultFunctionConfig is the default configuration for new function.
var DefaultFunctionConfig = FunctionConfig{
	Enabled: true,
}

var always = func(_ *common.Config) bool {
	return true
}

var isLogstash = func(cfg *common.Config) bool {
	return isOutput(cfg, "logstash")
}

var isElasticsearch = func(cfg *common.Config) bool {
	return isOutput(cfg, "elasticsearch")
}

func isOutput(cfg *common.Config, name string) bool {
	outputCfg, err := cfg.Child("output", -1)
	if err != nil {
		return false
	}
	return outputCfg.HasField(name)
}

type functionName string

func (f *functionName) Unpack(s string) error {
	if !functionRE.MatchString(s) {
		return fmt.Errorf(
			"invalid name: '%s', name must match [a-zA-Z0-9-] and be at most 140 characters",
			s,
		)
	}
	*f = functionName(s)
	return nil
}

func (f *functionName) String() string {
	return string(*f)
}

// Validate enforces that function names are unique.
func (p *ProviderConfig) Validate() error {
	names := make(map[functionName]bool)
	for _, rawfn := range p.Functions {
		fc := FunctionConfig{}
		rawfn.Unpack(&fc)

		if !fc.Enabled {
			return nil
		}

		if _, found := names[fc.Name]; found {
			return fmt.Errorf("function name '%s' already exist, name must be unique", fc.Name)
		}

		names[fc.Name] = true
	}
	return nil
}
