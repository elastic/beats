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

//go:build linux || darwin || windows

package add_docker_metadata

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/pkg/autodiscover/docker"
)

// Config for docker processor.
type Config struct {
	Host                string            `config:"host"`                           // Docker socket (UNIX or TCP socket).
	TLS                 *docker.TLSConfig `config:"ssl"`                            // TLS settings for connecting to Docker.
	Fields              []string          `config:"match_fields"`                   // A list of fields to match a container ID.
	MatchSource         bool              `config:"match_source"`                   // Match container ID from a log path present in source field.
	MatchShortID        bool              `config:"match_short_id"`                 // Match to container short ID from a log path present in source field.
	SourceIndex         int               `config:"match_source_index"`             // Index in the source path split by / to look for container ID.
	MatchPIDs           []string          `config:"match_pids"`                     // A list of fields containing process IDs (PIDs).
	HostFS              string            `config:"hostfs"`                         // Specifies the mount point of the host’s filesystem for use in monitoring a host from within a container.
	DeDot               bool              `config:"labels.dedot"`                   // If set to true, replace dots in labels with `_`.
	WaitMetadata        bool              `config:"wait_for_metadata"`              // Block initialization when Docker is unavailable until metadata is available.
	WaitMetadataTimeout time.Duration     `config:"wait_for_metadata_timeout"`      // Maximum time to wait for Docker metadata, 0 means wait indefinitely.
	WaitMetadataRetry   time.Duration     `config:"wait_for_metadata_retry_period"` // How long to wait between retries.

	// Annotations are kept after container is killed, until they haven't been
	// accessed for a full `cleanup_timeout`:
	CleanupTimeout time.Duration `config:"cleanup_timeout"`
}

func defaultConfig() Config {
	return Config{
		Host:                "unix:///var/run/docker.sock",
		MatchSource:         true,
		SourceIndex:         4, // Use 4 to match the CID in /var/lib/docker/containers/<container_id>/*.log.
		MatchPIDs:           []string{"process.pid", "process.parent.pid"},
		DeDot:               true,
		WaitMetadata:        false,
		WaitMetadataTimeout: 30 * time.Second,
		WaitMetadataRetry:   10 * time.Second,
	}
}

func (c Config) Validate() error {
	if c.WaitMetadataTimeout < 0 {
		return fmt.Errorf("wait_for_metadata_timeout must be zero or greater (zero means wait indefinitely)")
	}

	if c.WaitMetadataRetry <= 0 {
		return fmt.Errorf("wait_for_metadata_retry_period must be greater than zero")
	}

	return nil
}
