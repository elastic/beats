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
// +build linux darwin windows

package docker

import (
	"net/http"
	"os"

	"github.com/docker/docker/client"

	"github.com/elastic/elastic-agent-libs/logp"
)

// NewClient builds and returns a new Docker client. On the first request the
// client will negotiate the API version with the server unless
// DOCKER_API_VERSION is set in the environment.
func NewClient(host string, httpClient *http.Client, httpHeaders map[string]string) (*client.Client, error) {
	log := logp.NewLogger("docker")

	opts := []client.Opt{
		client.WithHost(host),
		client.WithHTTPClient(httpClient),
		client.WithHTTPHeaders(httpHeaders),
	}

	version := os.Getenv("DOCKER_API_VERSION")
	if version != "" {
		log.Debugf("Docker client will use API version %v as set by the DOCKER_API_VERSION environment variable.", version)
		opts = append(opts, client.WithVersion(version))
	} else {
		log.Debug("Docker client will negotiate the API version on the first request.")
		opts = append(opts, client.WithAPIVersionNegotiation())
	}

	return client.NewClientWithOpts(opts...)
}
