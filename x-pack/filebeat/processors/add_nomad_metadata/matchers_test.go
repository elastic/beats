// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package add_nomad_metadata

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const allocID = "43205e0e-3d55-f561-83cb-bed15e23b862"

func TestLogsPathMatcherEmpty(t *testing.T) {
	cfgLogsPath := "" // use the default matcher configuration
	var path string
	if runtime.GOOS == "windows" {
		path = `C:\ProgramData\Nomad`
	} else {
		path = "/var/lib/nomad"
	}
	expectedResult := ""
	executeTest(t, cfgLogsPath, path, expectedResult)
}

func TestLogsPathMatcherWithAllocation(t *testing.T) {
	var cfgLogsPath, path string
	if runtime.GOOS == "windows" {
		cfgLogsPath = `C:\ProgramData\Nomad\alloc`
		path = `C:\ProgramData\Nomad\alloc\43205e0e-3d55-f561-83cb-bed15e23b862\alloc\logs\teb-booking-gateway-prod.stdout.94`
	} else {
		cfgLogsPath = "/appdata/nomad/alloc/"
		path = "/appdata/nomad/alloc/43205e0e-3d55-f561-83cb-bed15e23b862/alloc/logs/teb-booking-gateway-prod.stdout.94"
	}

	executeTest(t, cfgLogsPath, path, allocID)
}

func executeTest(t *testing.T, cfgLogsPath string, source string, expectedResult string) {
	cfg := conf.NewConfig()
	if cfgLogsPath != "" {
		cfg.SetString("logs_path", -1, cfgLogsPath)
	}

	logMatcher, err := newLogsPathMatcher(*cfg)
	assert.Nil(t, err)

	input := mapstr.M{
		"log": mapstr.M{
			"file": mapstr.M{
				"path": source,
			},
		},
	}

	output := logMatcher.MetadataIndex(input)

	assert.Equal(t, expectedResult, output)
}
