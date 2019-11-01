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

package emptydir

import (
	"fmt"

	"github.com/elastic/beats/filebeat/autodiscover/builder/hints"
	"github.com/elastic/beats/libbeat/autodiscover"
	"github.com/elastic/beats/libbeat/autodiscover/builder"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
)

const (
	emptyDir = "emptydir"
	key      = "hints"
)

func init() {
	autodiscover.Registry.AddBuilder(emptyDir, NewEmptyDirBuilder)
}

type logPath struct {
	rootDir        string
	key            string
	defaultEnabled bool
	logBuilder     autodiscover.Builder
}

//NewEmptyDirBuilder creates an autodiscover Builder that can understand emptydir hints.
func NewEmptyDirBuilder(cfg *common.Config) (autodiscover.Builder, error) {
	config := defaultConfig()
	err := cfg.Unpack(&config)

	if err != nil {
		return nil, fmt.Errorf("unable to unpack config due to error: %v", err)
	}

	newCfg := common.MapStr{
		"key":            config.Key,
		"default_config": config.DefaultConfig,
		"type":           key,
	}

	newC, _ := common.NewConfigFrom(&newCfg)
	logBuilder, err := hints.NewLogHints(newC)
	if err != nil {
		return nil, fmt.Errorf("unable to generate logs builder due to error: %v", err)
	}

	defaultEnabled := false
	if newC.Enabled() == true {
		defaultEnabled = true
	}

	return &logPath{config.RootDir, config.Key, defaultEnabled, logBuilder}, nil
}

//CreateConfig creates input configs basede on emptydir hints.
func (l *logPath) CreateConfig(event bus.Event) []*common.Config {
	var config []*common.Config

	host, _ := event["host"].(string)
	if host == "" {
		return config
	}

	var hints common.MapStr
	hIface, ok := event["hints"]
	if ok {
		hints, _ = hIface.(common.MapStr)
	}
	if l.defaultEnabled == false && builder.IsEnabled(hints, l.key) == false {
		return config
	}

	e := common.MapStr(event)

	id, _ := e.GetValue("kubernetes.pod.uid")
	if id == nil {
		return config
	}

	config = append(config, l.getInputConfigs(hints, host, id.(string))...)
	return config
}

func (l *logPath) getInputConfigs(hints common.MapStr, host, id string) []*common.Config {
	var config []*common.Config

	// Extract all entries that are stored under emptydir
	cfgMap := builder.GetHintMapStr(hints, l.key, emptyDir)
	for k, v := range cfgMap {
		// For each empty dir, get all prospector configurations
		hints := common.MapStr{
			l.key: common.MapStr{
				k: v,
			},
		}
		configs := builder.GetConfigs(hints, l.key, k)
		for _, cfg := range configs {
			hints := common.MapStr{
				l.key: cfg,
			}

			files := builder.GetHintAsList(hints, l.key, "paths")
			// If no paths are configured then
			if len(files) == 0 {
				continue
			}

			var paths []string
			for _, file := range files {
				paths = append(paths, fmt.Sprintf("%s%s/volumes/kubernetes.io~empty-dir/%s%s", l.rootDir, id, k, file))
			}

			// If there are no paths then don't generate a config.
			if len(paths) == 0 {
				continue
			}

			e := bus.Event{
				"host":  host,
				"hints": hints,
				"paths": paths,
			}

			cfgs := l.logBuilder.CreateConfig(e)
			config = append(config, cfgs...)
		}
	}

	return config
}
