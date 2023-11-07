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

package beater

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestMakeESClient(t *testing.T) {
	t.Run("should not modify the timeout setting from original config", func(t *testing.T) {
		origTimeout := 90
		origCfg, _ := conf.NewConfigFrom(map[interface{}]interface{}{
			"hosts":    []string{"http://localhost:9200"},
			"username": "anyuser",
			"password": "anypwd",
			"timeout":  origTimeout,
		})
		anyAttempt := 1
		anyDuration := 1 * time.Second

		_, _ = makeESClient(origCfg, anyAttempt, anyDuration)

		timeout, err := origCfg.Int("timeout", -1)
		require.NoError(t, err)
		assert.EqualValues(t, origTimeout, timeout)
	})
}
