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

package template

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestConfig(t *testing.T) {
	testdata := []struct {
		cfg      common.MapStr
		template Config
		err      string
		name     string
	}{
		{name: "invalid", cfg: nil, template: Config{}, err: "template configuration requires a name"},
		{name: "default config", cfg: common.MapStr{"name": "beat"}, template: Config{Enabled: true, Name: "beat", Pattern: "beat*"}},
	}
	for _, td := range testdata {
		t.Run(td.name, func(t *testing.T) {
			cfg, err := common.NewConfigFrom(td.cfg)
			require.NoError(t, err)
			var tmp Config
			err = cfg.Unpack(&tmp)
			if td.err == "" {
				assert.NoError(t, err)
				assert.Equal(t, td.template, tmp)
			} else if assert.Error(t, err) {
				assert.True(t, strings.Contains(err.Error(), td.err), fmt.Sprintf("Error `%s` doesn't contain expected error string", err.Error()))
			}
		})
	}

}
