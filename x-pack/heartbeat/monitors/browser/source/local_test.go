// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux || synthetics

package source

import (
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestLocalSourceValidate(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	fixtureDir := path.Join(filepath.Dir(filename), "fixtures/todos")
	tests := []struct {
		name string
		cfg  mapstr.M
		err  error
	}{
		{"valid", mapstr.M{
			"path": fixtureDir,
		}, ErrLocalUnsupportedType},
		{"invalid", mapstr.M{
			"path": "/not/a/path",
		}, ErrLocalUnsupportedType},
		{"nopath", mapstr.M{}, ErrLocalUnsupportedType},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := dummyLocal(tt.cfg)
			require.Error(t, err)
			require.Regexp(t, tt.err, err)
		})
	}
}

func dummyLocal(conf map[string]interface{}) (*LocalSource, error) {
	zus := &LocalSource{}
	y, _ := yaml.Marshal(conf)
	c, err := config.NewConfigWithYAML(y, string(y))
	if err != nil {
		return nil, err
	}
	err = c.Unpack(zus)
	return zus, err
}
