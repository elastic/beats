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

package container

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/filebeat/input/log"
	"github.com/elastic/beats/libbeat/common"

	"github.com/pkg/errors"
)

func init() {
	err := input.Register("container", NewInput)
	if err != nil {
		panic(err)
	}
}

// NewInput creates a new container input
func NewInput(
	cfg *common.Config,
	outletFactory channel.Connector,
	context input.Context,
) (input.Input, error) {
	// Wrap log input with custom docker settings
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrap(err, "reading container input config")
	}

	if err := validateStream(config.Stream); err != nil {
		return nil, err
	}

	if err := validateFormat(config.Format); err != nil {
		return nil, err
	}

	// Set partial line joining to true (both json-file and CRI)
	if err := cfg.SetBool("docker-json.partial", -1, true); err != nil {
		return nil, errors.Wrap(err, "update input config")
	}

	if err := cfg.SetBool("docker-json.cri_flags", -1, true); err != nil {
		return nil, errors.Wrap(err, "update input config")
	}

	// Allow stream selection (stdout/stderr/all)
	if err := cfg.SetString("docker-json.stream", -1, config.Stream); err != nil {
		return nil, errors.Wrap(err, "update input config")
	}

	if err := cfg.SetString("docker-json.format", -1, config.Format); err != nil {
		return nil, errors.Wrap(err, "update input config")
	}

	// Set symlinks to true as CRI-O paths could point to symlinks instead of the actual path.
	if err := cfg.SetBool("symlinks", -1, true); err != nil {
		return nil, errors.Wrap(err, "update input config")
	}

	// Add stream to meta to ensure different state per stream
	if config.Stream != "all" {
		if context.Meta == nil {
			context.Meta = map[string]string{}
		}
		context.Meta["stream"] = config.Stream
	}

	return log.NewInput(cfg, outletFactory, context)
}

func validateStream(val string) error {
	if stringInSlice(val, []string{"all", "stdout", "stderr"}) {
		return nil
	}

	return fmt.Errorf("Invalid value for stream: %s, supported values are: all, stdout, stderr", val)
}

func validateFormat(val string) error {
	val = strings.ToLower(val)
	if stringInSlice(val, []string{"auto", "docker", "cri"}) {
		return nil
	}

	return fmt.Errorf("Invalid value for format: %s, supported values are: auto, docker, cri", val)
}

func stringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}
