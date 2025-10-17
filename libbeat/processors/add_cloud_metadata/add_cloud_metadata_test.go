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

package add_cloud_metadata

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func Test_addCloudMetadata_String(t *testing.T) {
	const timeout = 100 * time.Millisecond
	cfg := conf.MustNewConfigFrom(map[string]any{
		"providers": []string{"openstack"},
		"host":      "fake:1234",
		"timeout":   timeout.String(),
	})
	p, err := New(cfg, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)
	assert.Eventually(t, func() bool { return p.String() == "add_cloud_metadata=<uninitialized>" }, timeout, 10*time.Millisecond)
	assert.Eventually(t, func() bool { return p.String() == "add_cloud_metadata={}" }, 2*timeout, 10*time.Millisecond)
}
