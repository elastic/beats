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

package filestream

import (
	"fmt"
	"regexp"
	"sync"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	externalMode = "external"
	internalMode = "internal"

	copytruncateStrategy = "copytruncate"
)

var experimentalWarning sync.Once

func newProspector(config config) (loginp.Prospector, error) {
	err := checkConfigCompatibility(config.FileWatcher, config.FileIdentity)
	if err != nil {
		return nil, err
	}

	filewatcher, err := newFileWatcher(config.Paths, config.FileWatcher)
	if err != nil {
		return nil, fmt.Errorf("error while creating filewatcher %w", err)
	}

	identifier, err := newFileIdentifier(config.FileIdentity, config.Reader.Parsers.Suffix)
	if err != nil {
		return nil, fmt.Errorf("error while creating file identifier: %w", err)
	}

	logp.L().
		With("filestream_id", config.ID).
		Debugf("file identity is set to %s", identifier.Name())

	fileprospector := fileProspector{
		filewatcher:         filewatcher,
		identifier:          identifier,
		ignoreOlder:         config.IgnoreOlder,
		ignoreInactiveSince: config.IgnoreInactive,
		cleanRemoved:        config.CleanRemoved,
		stateChangeCloser:   config.Close.OnStateChange,
	}
	if config.Rotation == nil {
		return &fileprospector, nil
	}

	rotationMethod := config.Rotation.Name()
	switch rotationMethod {
	case "":
		return &fileprospector, nil

	case internalMode:
		return nil, fmt.Errorf("not implemented: internal log rotation")

	case externalMode:
		externalConfig := config.Rotation.Config()
		cfg := rotationConfig{}
		err := externalConfig.Unpack(&cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack configuration of external rotation: %w", err)
		}
		strategy := cfg.Strategy.Name()
		switch strategy {
		case copytruncateStrategy:
			experimentalWarning.Do(func() {
				cfgwarn.Experimental("rotation.external.copytruncate is used.")
			})

			cpCfg := &copyTruncateConfig{}
			err = cfg.Strategy.Config().Unpack(&cpCfg)
			if err != nil {
				return nil, fmt.Errorf("failed to unpack configuration of external copytruncate rotation: %w", err)
			}
			suffix, err := regexp.Compile(cpCfg.SuffixRegex)
			if err != nil {
				return nil, fmt.Errorf("invalid suffix regex for copytruncate rotation")
			}
			fileprospector.stateChangeCloser.Renamed = false
			return &copyTruncateFileProspector{
				fileprospector,
				suffix,
				newRotatedFilestreams(cpCfg),
			}, nil
		default:
		}
		return nil, fmt.Errorf("no such external rotation strategy: %s", strategy)

	default:
	}
	return nil, fmt.Errorf("no such rotation method: %s", rotationMethod)
}

func checkConfigCompatibility(fileWatcher, fileIdentifier *conf.Namespace) error {
	var fwCfg struct {
		Fingerprint struct {
			Enabled bool `config:"enabled"`
		} `config:"fingerprint"`
	}

	if fileWatcher != nil && fileIdentifier != nil && fileIdentifier.Name() == fingerprintName {
		err := fileWatcher.Config().Unpack(&fwCfg)
		if err != nil {
			return fmt.Errorf("failed to parse file watcher configuration: %w", err)
		}
		if !fwCfg.Fingerprint.Enabled {
			return fmt.Errorf("fingerprint file identity can be used only when fingerprint is enabled in the scanner")
		}
	}

	return nil
}
