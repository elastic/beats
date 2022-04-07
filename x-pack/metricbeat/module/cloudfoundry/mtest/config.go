// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package mtest

import (
	"testing"

	"github.com/elastic/beats/v8/libbeat/beat"
	cftest "github.com/elastic/beats/v8/x-pack/libbeat/common/cloudfoundry/test"
)

func GetConfig(t *testing.T, metricset string) map[string]interface{} {
	t.Helper()

	config := cftest.GetConfigFromEnv(t)
	config["module"] = "cloudfoundry"
	config["metricsets"] = []string{metricset}

	return config
}

// CleanFields removes fields that can contain data of real deployments
func CleanFields(e beat.Event) {
	e.Fields.Delete("cloudfoundry.tags.system_domain")
}
