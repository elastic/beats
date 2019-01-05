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
	"net/http"
	"os"

	"github.com/docker/docker/api"
	"github.com/docker/docker/api/types/versions"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"

	"github.com/elastic/beats/libbeat/logp"
)

// NewClient builds and returns a new Docker client
// It uses version 1.30 by default, and negotiates it with the server so it is downgraded if 1.30 is too high
func NewClient(host string, httpClient *http.Client, httpHeaders map[string]string) (*client.Client, error) {
	version := os.Getenv("DOCKER_API_VERSION")
	if version == "" {
		version = api.DefaultVersion
	}

	c, err := client.NewClient(host, version, httpClient, nil)
	if err != nil {
		return c, err
	}

	if os.Getenv("DOCKER_API_VERSION") == "" {
		logp.Debug("docker", "Negotiating client version")
		ping, err := c.Ping(context.Background())
		if err != nil {
			logp.Debug("docker", "Failed to perform ping: %s", err)
		}

		// try a really old version, before versioning headers existed
		if ping.APIVersion == "" {
			ping.APIVersion = "1.22"
		}

		// if server version is lower than the client version, downgrade
		if versions.LessThan(ping.APIVersion, version) {
			c.UpdateClientVersion(ping.APIVersion)
		}
	}

	logp.Debug("docker", "Client version set to %s", c.ClientVersion())

	return c, nil
}
