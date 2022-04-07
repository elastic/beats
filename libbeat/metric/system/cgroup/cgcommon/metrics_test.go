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

package cgcommon

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/opt"
)

func TestPressure(t *testing.T) {
	v2Path := "../testdata/docker/sys/fs/cgroup/system.slice/docker-1c8fa019edd4b9d4b2856f4932c55929c5c118c808ed5faee9a135ca6e84b039.scope"

	pressureData, err := GetPressure(filepath.Join(v2Path, "io.pressure"))
	assert.NoError(t, err, "error in getPressure")

	goodP := map[string]Pressure{
		"some": {
			Ten:          opt.Pct{Pct: 3.00},
			Sixty:        opt.Pct{Pct: 2.10},
			ThreeHundred: opt.Pct{Pct: 4.00},
			Total:        opt.UintWith(1154482),
		},
		"full": {
			Ten:          opt.Pct{Pct: 10},
			Sixty:        opt.Pct{Pct: 30},
			ThreeHundred: opt.Pct{Pct: 0.5},
			Total:        opt.UintWith(1154482),
		},
	}

	assert.Equal(t, goodP, pressureData, "pressure stats not equal")
}
