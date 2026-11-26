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

package resolve

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/shirou/gopsutil/v4/common"
	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	ctx := Context(context.Background(), NewTestResolver("/hostfs"))
	env, ok := ctx.Value(common.EnvKey).(common.EnvMap)
	if !assert.True(t, ok, "context must carry a gopsutil EnvMap") {
		return
	}
	assert.Equal(t, common.EnvMap{
		common.HostProcEnvKey: filepath.Join("/hostfs", "proc"),
		common.HostSysEnvKey:  filepath.Join("/hostfs", "sys"),
		common.HostEtcEnvKey:  filepath.Join("/hostfs", "etc"),
	}, env, "all host paths must be resolved through the resolver")
}
