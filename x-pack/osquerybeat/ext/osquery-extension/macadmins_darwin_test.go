// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build darwin

package main

import (
	"testing"

	"github.com/osquery/osquery-go/plugin/table"

	// Verify macadmins darwin-only extension imports work
	"github.com/macadmins/osquery-extension/tables/alt_system_info"
	"github.com/macadmins/osquery-extension/tables/authdb"
	"github.com/macadmins/osquery-extension/tables/energyimpact"
	"github.com/macadmins/osquery-extension/tables/filevaultusers"
	"github.com/macadmins/osquery-extension/tables/localnetworkpermissions"
	macosprofiles "github.com/macadmins/osquery-extension/tables/macos_profiles"
	"github.com/macadmins/osquery-extension/tables/macosrsr"
	"github.com/macadmins/osquery-extension/tables/mdm"
	"github.com/macadmins/osquery-extension/tables/munki"
	"github.com/macadmins/osquery-extension/tables/networkquality"
	"github.com/macadmins/osquery-extension/tables/pendingappleupdates"
	"github.com/macadmins/osquery-extension/tables/sofa"
	"github.com/macadmins/osquery-extension/tables/unifiedlog"
	"github.com/macadmins/osquery-extension/tables/wifi_network"
)

// TestMacadminsDarwinTablesImport verifies that the macadmins darwin-only
// extension tables can be imported and have the expected function signatures.
// This is a build-time verification test.
func TestMacadminsDarwinTablesImport(t *testing.T) {
	// Test that darwin-only tables have the expected function signatures
	tables := []struct {
		name    string
		columns func() []table.ColumnDefinition
	}{
		{"energy_impact", energyimpact.EnergyImpactColumns},
		{"filevault_users", filevaultusers.FileVaultUsersColumns},
		{"local_network_permissions", localnetworkpermissions.LocalNetworkPermissionsColumns},
		{"macos_profiles", macosprofiles.MacOSProfilesColumns},
		{"mdm", mdm.MDMInfoColumns},
		{"munki_info", munki.MunkiInfoColumns},
		{"munki_installs", munki.MunkiInstallsColumns},
		{"network_quality", networkquality.NetworkQualityColumns},
		{"pending_apple_updates", pendingappleupdates.PendingAppleUpdatesColumns},
		{"macadmins_unified_log", unifiedlog.UnifiedLogColumns},
		{"macos_rsr", macosrsr.MacOSRsrColumns},
		{"sofa_security_release_info", sofa.SofaSecurityReleaseInfoColumns},
		{"sofa_unpatched_cves", sofa.SofaUnpatchedCVEsColumns},
		{"authdb", authdb.AuthDBColumns},
		{"wifi_network", wifi_network.WifiNetworkColumns},
		{"alt_system_info", alt_system_info.AltSystemInfoColumns},
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
