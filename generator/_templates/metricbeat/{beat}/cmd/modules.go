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

package cmd

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/cmd"
)

// BuildModulesManager adds support for modules management to a beat
func BuildModulesManager(beat *beat.Beat) (cmd.ModulesManager, error) {
	config := beat.BeatConfig

	glob, err := config.String("config.modules.path", -1)
	if err != nil {
		return nil, errors.Errorf("modules management requires '%s.config.modules.path' setting", Name)
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
