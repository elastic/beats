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

package docker

import (
	"fmt"
	"path"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/filebeat/input/log"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/pkg/errors"
)

func init() {
	err := input.Register("docker", NewInput)
	if err != nil {
		panic(err)
	}
}

// NewInput creates a new docker input
func NewInput(
	cfg *common.Config,
	outletFactory channel.Connector,
	context input.Context,
) (input.Input, error) {
	logger := logp.NewLogger("docker")

	// Wrap log input with custom docker settings
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrap(err, "reading docker input config")
	}

	// Docker input should make sure that no callers should ever pass empty strings as container IDs or paths
	// Hence we explicitly make sure that we catch such things and print stack traces in the event of
	// an invocation so that it can be fixed.
	var ids, paths []string
	for _, p := range config.Containers.Paths {
		if p != "" {
			paths = append(paths, p)
		} else {
			logger.Error("Docker paths can't be empty for Docker input config")
			logger.Debugw("Empty path for Docker logfile was received", logp.Stack("stacktrace"))
		}
	}

	for _, containerID := range config.Containers.IDs {
		if containerID != "" {
			ids = append(ids, containerID)
		} else {
			logger.Error("Docker container ID can't be empty for Docker input config")
			logger.Debugw("Empty docker container ID was received", logp.Stack("stacktrace"))
		}
	}

	if len(ids) == 0 && len(paths) == 0 {
		return nil, errors.New("Docker input requires at least one entry under 'containers.ids' or 'containers.paths'")
	}

	// IDs + Path and Paths are mutually exclusive. Ensure that only one of them are set in a given configuration
	if len(ids) != 0 && len(paths) != 0 {
		return nil, errors.New("can not provide both 'containers.ids' and 'containers.paths' in the same input config")
	}

	if len(ids) != 0 {
		for idx, containerID := range ids {
			cfg.SetString("paths", idx, path.Join(config.Containers.Path, containerID, "*.log"))
		}
	} else {
		for idx, p := range paths {
			cfg.SetString("paths", idx, p)
		}
	}

	if err := checkStream(config.Containers.Stream); err != nil {
		return nil, err
	}

	if err := cfg.SetString("docker-json.stream", -1, config.Containers.Stream); err != nil {
		return nil, errors.Wrap(err, "update input config")
	}

	if err := cfg.SetBool("docker-json.partial", -1, config.Partial); err != nil {
		return nil, errors.Wrap(err, "update input config")
	}

	if err := cfg.SetBool("docker-json.cri_flags", -1, config.CRIFlags); err != nil {
		return nil, errors.Wrap(err, "update input config")
	}

	if err := cfg.SetBool("docker-json.force_cri_logs", -1, config.CRIForce); err != nil {
		return nil, errors.Wrap(err, "update input config")
	}

	if len(paths) != 0 {
		// Set symlinks to true as CRI-O paths could point to symlinks instead of the actual path.
		if err := cfg.SetBool("symlinks", -1, true); err != nil {
			return nil, errors.Wrap(err, "update input config")
		}
	}

	// Add stream to meta to ensure different state per stream
	if config.Containers.Stream != "all" {
		if context.Meta == nil {
			context.Meta = map[string]string{}
		}
		context.Meta["stream"] = config.Containers.Stream
	}

	return log.NewInput(cfg, outletFactory, context)
}

func checkStream(val string) error {
	for _, s := range []string{"all", "stdout", "stderr"} {
		if s == val {
			return nil
		}
	}

	return fmt.Errorf("Invalid value for containers.stream: %s, supported values are: all, stdout, stderr", val)
}
