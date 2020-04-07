// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"io/ioutil"
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
	tmp, err := ioutil.TempDir("", "watch")
	require.NoError(t, err)
	defer os.RemoveAll(tmp)

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
	ioutil.WriteFile(out, b, 0600)
}
