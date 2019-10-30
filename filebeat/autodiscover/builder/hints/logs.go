// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package hints

import (
	"fmt"
	"regexp"

	"github.com/elastic/beats/filebeat/fileset"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/libbeat/autodiscover"
	"github.com/elastic/beats/libbeat/autodiscover/builder"
	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/logp"
)

func init() {
	autodiscover.Registry.AddBuilder("hints", NewLogHints)
}

const (
	multiline    = "multiline"
	includeLines = "include_lines"
	excludeLines = "exclude_lines"
	processors   = "processors"
	json         = "json"
)

// validModuleNames to sanitize user input
var validModuleNames = regexp.MustCompile("[^a-zA-Z0-9\\_\\-]+")

type logHints struct {
	config   *config
	registry *fileset.ModuleRegistry
}

// NewLogHints builds a log hints builder
func NewLogHints(cfg *common.Config) (autodiscover.Builder, error) {
	config := defaultConfig()
	err := cfg.Unpack(&config)

	if err != nil {
		return nil, fmt.Errorf("unable to unpack hints config due to error: %v", err)
	}

	moduleRegistry, err := fileset.NewModuleRegistry([]*common.Config{}, "", false)
	if err != nil {
		return nil, err
	}

	return &logHints{&config, moduleRegistry}, nil
}

// Create config based on input hints in the bus event
func (l *logHints) CreateConfig(event bus.Event) []*common.Config {
	var hints common.MapStr
	hIface, ok := event["hints"]
	if ok {
		hints, _ = hIface.(common.MapStr)
	}

	inputConfig := l.getInputs(hints)

	// If default config is disabled return nothing unless it's explicty enabled
	if !l.config.DefaultConfig.Enabled() && !builder.IsEnabled(hints, l.config.Key) {
		logp.Debug("hints.builder", "default config is disabled: %+v", event)
		return []*common.Config{}
	}

	// If explicty disabled, return nothing
	if builder.IsDisabled(hints, l.config.Key) {
		logp.Debug("hints.builder", "logs disabled by hint: %+v", event)
		return []*common.Config{}
	}

	// Clone original config, enable it if disabled
	config, _ := common.NewConfigFrom(l.config.DefaultConfig)
	config.Remove("enabled", -1)

	host, _ := event["host"].(string)
	if host == "" {
		return []*common.Config{}
	}

	if inputConfig != nil {
		configs := []*common.Config{}
		for _, cfg := range inputConfig {
			if config, err := common.NewConfigFrom(cfg); err == nil {
				configs = append(configs, config)
			}
		}
		logp.Debug("hints.builder", "generated config %+v", configs)
		// Apply information in event to the template to generate the final config
		return template.ApplyConfigTemplate(event, configs)
	}

	tempCfg := common.MapStr{}
	mline := l.getMultiline(hints)
	if len(mline) != 0 {
		tempCfg.Put(multiline, mline)
	}
	if ilines := l.getIncludeLines(hints); len(ilines) != 0 {
		tempCfg.Put(includeLines, ilines)
	}
	if elines := l.getExcludeLines(hints); len(elines) != 0 {
		tempCfg.Put(excludeLines, elines)
	}

	if procs := l.getProcessors(hints); len(procs) != 0 {
		tempCfg.Put(processors, procs)
	}

	if jsonOpts := l.getJSONOptions(hints); len(jsonOpts) != 0 {
		tempCfg.Put(json, jsonOpts)
	}
	// Merge config template with the configs from the annotations
	if err := config.Merge(tempCfg); err != nil {
		logp.Debug("hints.builder", "config merge failed with error: %v", err)
		return []*common.Config{config}
	}

	module := l.getModule(hints)
	if module != "" {
		moduleConf := map[string]interface{}{
			"module": module,
		}

		filesets := l.getFilesets(hints, module)
		for fileset, conf := range filesets {
			filesetConf, _ := common.NewConfigFrom(config)

			if inputType, _ := filesetConf.String("type", -1); inputType == harvester.ContainerType {
				filesetConf.SetString("stream", -1, conf.Stream)
			} else {
				filesetConf.SetString("containers.stream", -1, conf.Stream)
			}

			moduleConf[fileset+".enabled"] = conf.Enabled
			moduleConf[fileset+".input"] = filesetConf

			logp.Debug("hints.builder", "generated config %+v", moduleConf)
		}
		config, _ = common.NewConfigFrom(moduleConf)
	}
	logp.Debug("hints.builder", "generated config %+v", config)

	// Apply information in event to the template to generate the final config
	return template.ApplyConfigTemplate(event, []*common.Config{config})
}

func (l *logHints) getMultiline(hints common.MapStr) common.MapStr {
	return builder.GetHintMapStr(hints, l.config.Key, multiline)
}

func (l *logHints) getIncludeLines(hints common.MapStr) []string {
	return builder.GetHintAsList(hints, l.config.Key, includeLines)
}

func (l *logHints) getExcludeLines(hints common.MapStr) []string {
	return builder.GetHintAsList(hints, l.config.Key, excludeLines)
}

func (l *logHints) getModule(hints common.MapStr) string {
	module := builder.GetHintString(hints, l.config.Key, "module")
	// for security, strip module name
	return validModuleNames.ReplaceAllString(module, "")
}

func (l *logHints) getInputs(hints common.MapStr) []common.MapStr {
	return builder.GetHintAsConfigs(hints, l.config.Key)
}

func (l *logHints) getProcessors(hints common.MapStr) []common.MapStr {
	return builder.GetProcessors(hints, l.config.Key)
}

func (l *logHints) getJSONOptions(hints common.MapStr) common.MapStr {
	return builder.GetHintMapStr(hints, l.config.Key, json)
}

type filesetConfig struct {
	Enabled bool
	Stream  string
}

// Return a map containing filesets -> enabled & stream (stdout, stderr, all)
func (l *logHints) getFilesets(hints common.MapStr, module string) map[string]*filesetConfig {
	var configured bool
	filesets := make(map[string]*filesetConfig)

	moduleFilesets, err := l.registry.ModuleFilesets(module)
	if err != nil {
		logp.Err("Error retrieving module filesets: %+v", err)
		return nil
	}

	for _, fileset := range moduleFilesets {
		filesets[fileset] = &filesetConfig{Enabled: false, Stream: "all"}
	}

	// If a single fileset is given, pass all streams to it
	fileset := builder.GetHintString(hints, l.config.Key, "fileset")
	if fileset != "" {
		if conf, ok := filesets[fileset]; ok {
			conf.Enabled = true
			configured = true
		}
	}

	// If fileset is defined per stream, return all of them
	for _, stream := range []string{"all", "stdout", "stderr"} {
		fileset := builder.GetHintString(hints, l.config.Key, "fileset."+stream)
		if fileset != "" {
			if conf, ok := filesets[fileset]; ok {
				conf.Enabled = true
				conf.Stream = stream
				configured = true
			}
		}
	}

	// No fileset defined, return defaults for the module, all streams to all filesets
	if !configured {
		for _, conf := range filesets {
			conf.Enabled = true
		}
	}

	return filesets
}
