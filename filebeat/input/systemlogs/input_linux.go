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

//go:build linux

package systemlogs

import (
	"fmt"

	"github.com/elastic/beats/v7/filebeat/input/journald"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	conf "github.com/elastic/elastic-agent-libs/config"
)

// configure checks whether the journald input must be created and
// delegates to journald.Configure if needed.
func configure(cfg *conf.C) ([]cursor.Source, cursor.Input, error) {
	jouranl, err := useJournald(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot decide between journald and files: %w", err)
	}

	if !jouranl {
		return nil, nil, v2.ErrUnknownInput
	}

	journaldCfg, err := toJournaldConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	return journald.Configure(journaldCfg)
}

func toJournaldConfig(cfg *conf.C) (*conf.C, error) {
	newCfg, err := cfg.Child("journald", -1)
	if err != nil {
		return nil, fmt.Errorf("cannot extract 'journald' block: %w", err)
	}

	if _, err := cfg.Remove("journald", -1); err != nil {
		return nil, err
	}

	if _, err := cfg.Remove("type", -1); err != nil {
		return nil, err
	}

	if _, err := cfg.Remove("files", -1); err != nil {
		return nil, err
	}

	if _, err := cfg.Remove("use_journald", -1); err != nil {
		return nil, err
	}

	if _, err := cfg.Remove("use_files", -1); err != nil {
		return nil, err
	}

	if err := newCfg.Merge(cfg); err != nil {
		return nil, err
	}

	if err := newCfg.SetString("type", -1, "journald"); err != nil {
		return nil, fmt.Errorf("cannot set 'type': %w", err)
	}

	if err := cfg.SetString("type", -1, pluginName); err != nil {
		return nil, fmt.Errorf("cannot set type back to '%s': %w", pluginName, err)
	}

	return newCfg, nil
}
