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

	"github.com/elastic/go-ucfg"

	"github.com/elastic/elastic-agent-autodiscover/bus"
	"github.com/elastic/elastic-agent-autodiscover/utils"

	"github.com/elastic/beats/v7/filebeat/fileset"
	"github.com/elastic/beats/v7/filebeat/harvester"
	"github.com/elastic/beats/v7/libbeat/autodiscover"
	"github.com/elastic/beats/v7/libbeat/autodiscover/providers/kubernetes"
	"github.com/elastic/beats/v7/libbeat/autodiscover/template"
	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func init() {
	err := autodiscover.Registry.AddBuilder("hints", NewLogHints)
	if err != nil {
		logp.Error(fmt.Errorf("could not add `hints` builder"))
	}
}

const (
	multiline    = "multiline"
	includeLines = "include_lines"
	excludeLines = "exclude_lines"
	processors   = "processors"
	json         = "json"
	pipeline     = "pipeline"
	ndjson       = "ndjson"
	parsers      = "parsers"
)

// validModuleNames to sanitize user input
var validModuleNames = regexp.MustCompile(`[^a-zA-Z0-9\\_\\-]+`)

type logHints struct {
	config   *config
	registry *fileset.ModuleRegistry
	log      *logp.Logger
}

// NewLogHints builds a log hints builder
func NewLogHints(cfg *conf.C) (autodiscover.Builder, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, fmt.Errorf("unable to unpack hints config due to error: %w", err)
	}

	moduleRegistry, err := fileset.NewModuleRegistry(nil, beat.Info{}, false, fileset.FilesetOverrides{})
	if err != nil {
		return nil, err
	}

	return &logHints{&config, moduleRegistry, logp.NewLogger("hints.builder")}, nil
}

// Create config based on input hints in the bus event
func (l *logHints) CreateConfig(event bus.Event, options ...ucfg.Option) []*conf.C {
	var hints mapstr.M
	if hintsIfc, found := event["hints"]; found {
		hints, _ = hintsIfc.(mapstr.M)
	}

	// Hint must be explicitly enabled when default_config sets enabled=false.
	if !l.config.DefaultConfig.Enabled() && !utils.IsEnabled(hints, l.config.Key) ||
		utils.IsDisabled(hints, l.config.Key) {
		l.log.Debugw("Hints config is not enabled.", "autodiscover.event", event)
		return nil
	}

	if inputConfig := l.getInputsConfigs(hints); inputConfig != nil {
		var configs []*conf.C
		for _, cfg := range inputConfig {
			if config, err := conf.NewConfigFrom(cfg); err == nil {
				configs = append(configs, config)
			} else {
				l.log.Warnw("Failed to create config from input.", "error", err)
			}
		}
		l.log.Debugf("Generated %d input configs from hint.", len(configs))
		// Apply information in event to the template to generate the final config
		return template.ApplyConfigTemplate(event, configs)
	}

	var configs []*conf.C //nolint:prealloc //breaks tests
	inputs := l.getInputs(hints)
	for _, h := range inputs {
		// Clone original config, enable it if disabled
		config, _ := conf.NewConfigFrom(l.config.DefaultConfig)
		_, err := config.Remove("enabled", -1)
		if err != nil {
			continue
		}

		inputType, _ := config.String("type", -1)
		tempCfg := mapstr.M{}

		if mline := l.getMultiline(h); len(mline) != 0 {
			if inputType == harvester.FilestreamType {
				// multiline options should be under multiline parser in filestream input
				parsersTempCfg := []mapstr.M{}
				mlineTempCfg := mapstr.M{}
				kubernetes.ShouldPut(mlineTempCfg, multiline, mline, l.log)
				parsersTempCfg = append(parsersTempCfg, mlineTempCfg)
				kubernetes.ShouldPut(tempCfg, parsers, parsersTempCfg, l.log)
			} else {
				kubernetes.ShouldPut(tempCfg, multiline, mline, l.log)
			}
		}
		if ilines := l.getIncludeLines(h); len(ilines) != 0 {
			kubernetes.ShouldPut(tempCfg, includeLines, ilines, l.log)
		}
		if elines := l.getExcludeLines(h); len(elines) != 0 {
			kubernetes.ShouldPut(tempCfg, excludeLines, elines, l.log)
		}

		if procs := l.getProcessors(h); len(procs) != 0 {
			kubernetes.ShouldPut(tempCfg, processors, procs, l.log)
		}

		if pip := l.getPipeline(h); len(pip) != 0 {
			kubernetes.ShouldPut(tempCfg, pipeline, pip, l.log)
		}

		if jsonOpts := l.getJSONOptions(h); len(jsonOpts) != 0 {
			if inputType == harvester.FilestreamType {
				// json options should be under ndjson parser in filestream input
				parsersTempCfg := []mapstr.M{}
				ndjsonTempCfg := mapstr.M{}
				kubernetes.ShouldPut(ndjsonTempCfg, ndjson, jsonOpts, l.log)
				parsersTempCfg = append(parsersTempCfg, ndjsonTempCfg)
				kubernetes.ShouldPut(tempCfg, parsers, parsersTempCfg, l.log)
			} else {
				kubernetes.ShouldPut(tempCfg, json, jsonOpts, l.log)
			}

		}
		// Merge config template with the configs from the annotations
		// AppendValues option is used to append arrays from annotations to existing arrays while merging
		if err := config.MergeWithOpts(tempCfg, ucfg.AppendValues); err != nil {
			l.log.Debugf("hints.builder", "config merge failed with error: %v", err)
			continue
		}
		module := l.getModule(hints)
		if module != "" {
			moduleConf := map[string]interface{}{
				"module": module,
			}

			filesets := l.getFilesets(hints, module)
			for fileset, cfg := range filesets {
				filesetConf, _ := conf.NewConfigFrom(config)
				if inputType == harvester.ContainerType {
					_ = filesetConf.SetString("stream", -1, cfg.Stream)
				} else if inputType == harvester.FilestreamType {
					filestreamContainerParser := map[string]interface{}{
						"container": map[string]interface{}{
							"stream": cfg.Stream,
							"format": "auto",
						},
					}
					parserCfg, _ := conf.NewConfigFrom(filestreamContainerParser)
					_ = filesetConf.SetChild("parsers", 0, parserCfg)
				} else {
					_ = filesetConf.SetString("containers.stream", -1, cfg.Stream)
				}

				moduleConf[fileset+".enabled"] = cfg.Enabled
				moduleConf[fileset+".input"] = filesetConf

				l.log.Debugf("hints.builder", "generated config %+v", moduleConf)
			}
			config, _ = conf.NewConfigFrom(moduleConf)
		}
		l.log.Debugf("hints.builder", "generated config %+v of logHints %+v", config, l)
		configs = append(configs, config)
	}
	// Apply information in event to the template to generate the final config
	return template.ApplyConfigTemplate(event, configs)
}

func (l *logHints) getMultiline(hints mapstr.M) mapstr.M {
	return utils.GetHintMapStr(hints, l.config.Key, multiline)
}

func (l *logHints) getIncludeLines(hints mapstr.M) []string {
	return utils.GetHintAsList(hints, l.config.Key, includeLines)
}

func (l *logHints) getExcludeLines(hints mapstr.M) []string {
	return utils.GetHintAsList(hints, l.config.Key, excludeLines)
}

func (l *logHints) getModule(hints mapstr.M) string {
	module := utils.GetHintString(hints, l.config.Key, "module")
	// for security, strip module name
	return validModuleNames.ReplaceAllString(module, "")
}

func (l *logHints) getInputsConfigs(hints mapstr.M) []mapstr.M {
	return utils.GetHintAsConfigs(hints, l.config.Key)
}

func (l *logHints) getProcessors(hints mapstr.M) []mapstr.M {
	return utils.GetProcessors(hints, l.config.Key)
}

func (l *logHints) getPipeline(hints mapstr.M) string {
	return utils.GetHintString(hints, l.config.Key, "pipeline")
}

func (l *logHints) getJSONOptions(hints mapstr.M) mapstr.M {
	return utils.GetHintMapStr(hints, l.config.Key, json)
}

type filesetConfig struct {
	Enabled bool
	Stream  string
}

// Return a map containing filesets -> enabled & stream (stdout, stderr, all)
func (l *logHints) getFilesets(hints mapstr.M, module string) map[string]*filesetConfig {
	var configured bool
	filesets := make(map[string]*filesetConfig)

	moduleFilesets, err := l.registry.ModuleAvailableFilesets(module)
	if err != nil {
		l.log.Errorf("Error retrieving module filesets: %+v", err)
		return nil
	}

	for _, fileset := range moduleFilesets {
		filesets[fileset] = &filesetConfig{Enabled: false, Stream: "all"}
	}

	// If a single fileset is given, pass all streams to it
	fileset := utils.GetHintString(hints, l.config.Key, "fileset")
	if fileset != "" {
		if conf, ok := filesets[fileset]; ok {
			conf.Enabled = true
			configured = true
		}
	}

	// If fileset is defined per stream, return all of them
	for _, stream := range []string{"all", "stdout", "stderr"} {
		fileset := utils.GetHintString(hints, l.config.Key, "fileset."+stream)
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

func (l *logHints) getInputs(hints mapstr.M) []mapstr.M {
	modules := utils.GetHintsAsList(hints, l.config.Key)
	var output []mapstr.M //nolint:prealloc //breaks tests

	for _, mod := range modules {
		output = append(output, mapstr.M{
			l.config.Key: mod,
		})
	}

	// Generate this so that no hints with completely valid templates work
	if len(output) == 0 {
		output = append(output, mapstr.M{
			l.config.Key: mapstr.M{},
		})
	}

	return output
}
