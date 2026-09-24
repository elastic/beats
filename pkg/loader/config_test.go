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

package loader

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestConfig(t *testing.T) {
	testToMapStr(t)
	testLoadFiles(t)
}

func TestInputsResolveNOOP(t *testing.T) {
	contents := map[string]interface{}{
		"outputs": map[string]interface{}{
			"default": map[string]interface{}{
				"type":     "elasticsearch",
				"hosts":    []interface{}{"127.0.0.1:9200"},
				"username": "elastic",
				"password": "changeme",
			},
		},
		"inputs": []interface{}{
			map[string]interface{}{
				"type": "logfile",
				"streams": []interface{}{
					map[string]interface{}{
						"paths": []interface{}{"/var/log/${host.name}"},
					},
				},
			},
		},
	}

	cfgPath := filepath.Join(t.TempDir(), "config.yml")
	dumpToYAML(t, cfgPath, contents)

	cfg, err := LoadFile(cfgPath)
	require.NoError(t, err)

	cfgData, err := cfg.ToMapStr()
	require.NoError(t, err)

	assert.Equal(t, contents, cfgData)
}

func testToMapStr(t *testing.T) {
	m := map[string]interface{}{
		"hello": map[string]interface{}{
			"what": "who",
		},
	}

	c := MustNewConfigFrom(m)
	nm, err := c.ToMapStr()
	require.NoError(t, err)

	assert.True(t, reflect.DeepEqual(m, nm))
}

func testLoadFiles(t *testing.T) {
	tmp := t.TempDir()

	f1 := filepath.Join(tmp, "1.yml")
	dumpToYAML(t, f1, map[string]interface{}{
		"hello": map[string]interface{}{
			"what": "1",
		},
	})

	f2 := filepath.Join(tmp, "2.yml")
	dumpToYAML(t, f2, map[string]interface{}{
		"hello": map[string]interface{}{
			"where": "2",
		},
	})

	f3 := filepath.Join(tmp, "3.yml")
	dumpToYAML(t, f3, map[string]interface{}{
		"super": map[string]interface{}{
			"awesome": "cool",
		},
	})

	c, err := LoadFiles(f1, f2, f3)
	require.NoError(t, err)

	r, err := c.ToMapStr()
	require.NoError(t, err)

	assert.Equal(t, map[string]interface{}{
		"hello": map[string]interface{}{
			"what":  "1",
			"where": "2",
		},
		"super": map[string]interface{}{
			"awesome": "cool",
		},
	}, r)
}

func dumpToYAML(t *testing.T, out string, in interface{}) {
	b, err := yaml.Marshal(in)
	require.NoError(t, err)
	err = os.WriteFile(out, b, 0600)
	require.NoError(t, err)
}
