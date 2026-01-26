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
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	externalMode = "external"
	internalMode = "internal"

	copytruncateStrategy = "copytruncate"
)

var experimentalWarning sync.Once

func newProspector(
	config config,
	log *logp.Logger,
	srci *loginp.SourceIdentifier) (loginp.Prospector, error) {

	logger := log.Named("filestream").With("id", config.ID)
	err := checkConfigCompatibility(config)
	if err != nil {
		return nil, err
	}

	identifier, err := newFileIdentifier(
		config.FileIdentity,
		config.Reader.Parsers.Suffix,
		logger)
	if err != nil {
		return nil, fmt.Errorf("error while creating file identifier: %w", err)
	}
	logger.Debugf("file identity is set to %s", identifier.Name())

	filewatcher, err := newFileWatcher(
		logger,
		config.Paths,
		config.FileWatcher,
		config.Compression,
		config.Delete.Enabled,
		identifier,
		srci,
	)
	if err != nil {
		return nil, fmt.Errorf("error while creating filewatcher %w", err)
	}

	fileprospector := fileProspector{
		filewatcher:         filewatcher,
		identifier:          identifier,
		ignoreOlder:         config.IgnoreOlder,
		ignoreInactiveSince: config.IgnoreInactive,
		cleanRemoved:        config.CleanRemoved,
		stateChangeCloser:   config.Close.OnStateChange,
		logger:              logger.Named("prospector"),
		takeOver:            config.TakeOver,
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
				log.Warn(cfgwarn.Experimental("rotation.external.copytruncate is used."))
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

func checkConfigCompatibility(config config) error {
	if config.FileIdentity != nil &&
		config.FileIdentity.Name() == fingerprintName &&
		!config.FileWatcher.Scanner.Fingerprint.Enabled {
		return fmt.Errorf("fingerprint file identity can be used only when fingerprint is enabled in the scanner")
	}

	return nil
}
