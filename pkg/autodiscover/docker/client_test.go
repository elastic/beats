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

//go:build integration && (linux || darwin || windows)

package docker

import (
	"context"
	"testing"

	"github.com/moby/moby/client"
	"github.com/moby/moby/client/pkg/versions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/pkg/logp/logptest"
)

func TestNewClient(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	c, err := NewClient(client.DefaultDockerHost, nil, nil, logger)
	assert.NoError(t, err)
	assert.NotNil(t, c)

	_, err = c.ContainerList(context.Background(), client.ContainerListOptions{})
	assert.NoError(t, err)

	t.Logf("Negotiated version: %s", c.ClientVersion())

	server, err := c.ServerVersion(context.Background(), client.ServerVersionOptions{})
	require.NoError(t, err)

	// Pin the oldest API version both the client and the daemon accept. Daemons
	// reject versions below their minimum, so a fixed old version would break
	// whenever the CI Docker is upgraded.
	pinned := client.MinAPIVersion
	if versions.GreaterThan(server.MinAPIVersion, pinned) {
		pinned = server.MinAPIVersion
	}
	if pinned == c.ClientVersion() {
		t.Skipf("daemon only supports API version %s, cannot test pinning a different version", pinned)
	}

	// Test we can hardcode version
	t.Setenv("DOCKER_API_VERSION", pinned)

	c, err = NewClient(client.DefaultDockerHost, nil, nil, logger)
	assert.NoError(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, pinned, c.ClientVersion(), "DOCKER_API_VERSION should override version negotiation")

	_, err = c.ContainerList(context.Background(), client.ContainerListOptions{})
	assert.NoError(t, err)
}
