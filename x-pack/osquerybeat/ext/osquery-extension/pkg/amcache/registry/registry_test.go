// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one

// or more contributor license agreements. Licensed under the Elastic License;

// you may not use this file except in compliance with the Elastic License.

//go:build windows

package registry

import (
	"reflect"
	"testing"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/testdata"
)

func TestLoadRegistry(t *testing.T) {
	filePath := testdata.GetTestHivePathOrFatal(t)
	registry, err := LoadRegistry(filePath)
	if err != nil {
		t.Fatalf("failed to load registry: %v", err)
	}
	if registry == nil {
		t.Fatalf("registry is nil")
	}
	if len(registry.Subkeys()) == 0 {
		t.Fatalf("registry has no subkeys")
	}
	if len(registry.Values()) == 0 {
		t.Fatalf("registry has no values")
	}
}
