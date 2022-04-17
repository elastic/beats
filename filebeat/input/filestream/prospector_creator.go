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

	loginp "github.com/menderesk/beats/v7/filebeat/input/filestream/internal/input-logfile"
	"github.com/menderesk/beats/v7/libbeat/common/cfgwarn"
)

const (
	externalMode = "external"
	internalMode = "internal"

	copytruncateStrategy = "copytruncate"
)

var experimentalWarning sync.Once

func newProspector(config config) (loginp.Prospector, error) {
	filewatcher, err := newFileWatcher(config.Paths, config.FileWatcher)
	if err != nil {
		return nil, fmt.Errorf("error while creating filewatcher %v", err)
	}

	identifier, err := newFileIdentifier(config.FileIdentity, getIdentifierSuffix(config))
	if err != nil {
		return nil, fmt.Errorf("error while creating file identifier: %v", err)
	}

	fileprospector := fileProspector{
		filewatcher:       filewatcher,
		identifier:        identifier,
		ignoreOlder:       config.IgnoreOlder,
		cleanRemoved:      config.CleanRemoved,
		stateChangeCloser: config.Close.OnStateChange,
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
			return nil, fmt.Errorf("failed to unpack configuration of external rotation: %+v", err)
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
				return nil, fmt.Errorf("failed to unpack configuration of external copytruncate rotation: %+v", err)
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

func getIdentifierSuffix(config config) string {
	return config.Reader.Parsers.Suffix
}
