// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build darwin || linux

package main

import (
	"testing"

	"github.com/osquery/osquery-go/plugin/table"

	// Verify macadmins extension imports work
	"github.com/macadmins/osquery-extension/tables/chromeuserprofiles"
	"github.com/macadmins/osquery-extension/tables/crowdstrike_falcon"
	"github.com/macadmins/osquery-extension/tables/fileline"
	"github.com/macadmins/osquery-extension/tables/puppet"
)

// TestMacadminsTablesImport verifies that the macadmins extension tables
// can be imported and have the expected function signatures.
// This is a build-time verification test.
func TestMacadminsTablesImport(t *testing.T) {
	// Test that cross-platform tables have the expected function signatures
	tables := []struct {
		name    string
		columns func() []table.ColumnDefinition
	}{
		{"puppet_info", puppet.PuppetInfoColumns},
		{"puppet_logs", puppet.PuppetLogsColumns},
		{"puppet_state", puppet.PuppetStateColumns},
		{"puppet_facts", puppet.PuppetFactsColumns},
		{"google_chrome_profiles", chromeuserprofiles.GoogleChromeProfilesColumns},
		{"file_lines", fileline.FileLineColumns},
		{"crowdstrike_falcon", crowdstrike_falcon.CrowdstrikeFalconColumns},
	}

	for _, tt := range tables {
		t.Run(tt.name, func(t *testing.T) {
			columns := tt.columns()
			if len(columns) == 0 {
				t.Errorf("table %s has no columns defined", tt.name)
			}
		})
	}
}
