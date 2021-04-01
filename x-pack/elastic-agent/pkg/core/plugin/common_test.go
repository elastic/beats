// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package plugin

import (
	"testing"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestRestartNeeded(t *testing.T) {
	tt := []struct {
		Name          string
		OldOutput     map[string]interface{}
		NewOutput     map[string]interface{}
		ShouldRestart bool

		ExpectedRestart bool
	}{
		{
			"same empty output",
			map[string]interface{}{},
			map[string]interface{}{},
			true,
			false,
		},
		{
			"same not empty output",
			map[string]interface{}{"output": map[string]interface{}{"username": "user", "password": "123456"}},
			map[string]interface{}{"output": map[string]interface{}{"username": "user", "password": "123456"}},
			true,
			false,
		},
		{
			"different empty output",
			map[string]interface{}{},
			map[string]interface{}{"output": map[string]interface{}{"username": "user", "password": "123456"}},
			true,
			false,
		},
		{
			"different not empty output",
			map[string]interface{}{"output": map[string]interface{}{"username": "user", "password": "123456"}},
			map[string]interface{}{"output": map[string]interface{}{"username": "user", "password": "s3cur3_Pa55;"}},
			true,
			true,
		},
		{
			"different not empty output no restart required",
			map[string]interface{}{"output": map[string]interface{}{"username": "user", "password": "123456"}},
			map[string]interface{}{"output": map[string]interface{}{"username": "user", "password": "s3cur3_Pa55;"}},
			false,
			false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			cf, err := newTestConfigFetcher(tc.OldOutput)
			require.NoError(t, err)
			s := testProgramSpec(tc.ShouldRestart)
			l, _ := logger.New("tst")

			IsRestartNeeded(l, s, cf, tc.NewOutput)
		})
	}
}

func newTestConfigFetcher(cfg map[string]interface{}) (*testConfigFetcher, error) {
	cfgStr, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, errors.New(err, errors.TypeApplication)
	}

	return &testConfigFetcher{cfg: string(cfgStr)}, nil
}

type testConfigFetcher struct {
	cfg string
}

func (f testConfigFetcher) Config() string { return f.cfg }

func testProgramSpec(restartOnOutput bool) program.Spec {
	return program.Spec{
		RestartOnOutputChange: restartOnOutput,
	}
}
