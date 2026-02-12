// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build darwin

package main

import (
	"context"

	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/client"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/hooks"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"

	// macadmins extension tables
	"github.com/macadmins/osquery-extension/tables/alt_system_info"
	"github.com/macadmins/osquery-extension/tables/authdb"
	"github.com/macadmins/osquery-extension/tables/chromeuserprofiles"
	"github.com/macadmins/osquery-extension/tables/crowdstrike_falcon"
	"github.com/macadmins/osquery-extension/tables/energyimpact"
	"github.com/macadmins/osquery-extension/tables/fileline"
	"github.com/macadmins/osquery-extension/tables/filevaultusers"
	"github.com/macadmins/osquery-extension/tables/localnetworkpermissions"
	macosprofiles "github.com/macadmins/osquery-extension/tables/macos_profiles"
	"github.com/macadmins/osquery-extension/tables/macosrsr"
	"github.com/macadmins/osquery-extension/tables/mdm"
	"github.com/macadmins/osquery-extension/tables/munki"
	"github.com/macadmins/osquery-extension/tables/networkquality"
	"github.com/macadmins/osquery-extension/tables/pendingappleupdates"
	"github.com/macadmins/osquery-extension/tables/puppet"
	"github.com/macadmins/osquery-extension/tables/sofa"
	"github.com/macadmins/osquery-extension/tables/unifiedlog"
	"github.com/macadmins/osquery-extension/tables/wifi_network"
)

// macadminsExtensionVersion is used for sofa user agent
const macadminsExtensionVersion = "1.3.2"

func RegisterTables(server *osquery.ExtensionManagerServer, log *logger.Logger, _ *hooks.HookManager, _ *client.ResilientClient) {
	socketPath := *socket

	// Build sofa options with user agent
	useragent := sofa.BuildUserAgent(macadminsExtensionVersion)
	sofaOpts := []sofa.Option{
		sofa.WithUserAgent(useragent),
	}

	// Cross-platform tables from macadmins extension
	server.RegisterPlugin(table.NewPlugin("puppet_info", puppet.PuppetInfoColumns(), puppet.PuppetInfoGenerate))
	log.Infof("Registered macadmins table: puppet_info")

	server.RegisterPlugin(table.NewPlugin("puppet_logs", puppet.PuppetLogsColumns(), puppet.PuppetLogsGenerate))
	log.Infof("Registered macadmins table: puppet_logs")

	server.RegisterPlugin(table.NewPlugin("puppet_state", puppet.PuppetStateColumns(), puppet.PuppetStateGenerate))
	log.Infof("Registered macadmins table: puppet_state")

	server.RegisterPlugin(table.NewPlugin("puppet_facts", puppet.PuppetFactsColumns(), puppet.PuppetFactsGenerate))
	log.Infof("Registered macadmins table: puppet_facts")

	server.RegisterPlugin(table.NewPlugin("google_chrome_profiles", chromeuserprofiles.GoogleChromeProfilesColumns(), chromeuserprofiles.GoogleChromeProfilesGenerate))
	log.Infof("Registered macadmins table: google_chrome_profiles")

	server.RegisterPlugin(table.NewPlugin("file_lines", fileline.FileLineColumns(), fileline.FileLineGenerate))
	log.Infof("Registered macadmins table: file_lines")

	// Linux/Darwin table
	server.RegisterPlugin(table.NewPlugin(
		"crowdstrike_falcon",
		crowdstrike_falcon.CrowdstrikeFalconColumns(),
		func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
			return crowdstrike_falcon.CrowdstrikeFalconGenerate(ctx, queryContext, socketPath)
		}))
	log.Infof("Registered macadmins table: crowdstrike_falcon")

	// Darwin-only tables from macadmins extension
	server.RegisterPlugin(table.NewPlugin("energy_impact", energyimpact.EnergyImpactColumns(), energyimpact.EnergyImpactGenerate))
	log.Infof("Registered macadmins table: energy_impact")

	server.RegisterPlugin(table.NewPlugin("filevault_users", filevaultusers.FileVaultUsersColumns(), filevaultusers.FileVaultUsersGenerate))
	log.Infof("Registered macadmins table: filevault_users")

	server.RegisterPlugin(table.NewPlugin("local_network_permissions", localnetworkpermissions.LocalNetworkPermissionsColumns(), localnetworkpermissions.LocalNetworkPermissionsGenerate))
	log.Infof("Registered macadmins table: local_network_permissions")

	server.RegisterPlugin(table.NewPlugin("macos_profiles", macosprofiles.MacOSProfilesColumns(), macosprofiles.MacOSProfilesGenerate))
	log.Infof("Registered macadmins table: macos_profiles")

	server.RegisterPlugin(table.NewPlugin("mdm", mdm.MDMInfoColumns(), mdm.MDMInfoGenerate))
	log.Infof("Registered macadmins table: mdm")

	server.RegisterPlugin(table.NewPlugin("munki_info", munki.MunkiInfoColumns(), munki.MunkiInfoGenerate))
	log.Infof("Registered macadmins table: munki_info")

	server.RegisterPlugin(table.NewPlugin("munki_installs", munki.MunkiInstallsColumns(), munki.MunkiInstallsGenerate))
	log.Infof("Registered macadmins table: munki_installs")

	server.RegisterPlugin(table.NewPlugin("network_quality", networkquality.NetworkQualityColumns(), networkquality.NetworkQualityGenerate))
	log.Infof("Registered macadmins table: network_quality")

	server.RegisterPlugin(table.NewPlugin("pending_apple_updates", pendingappleupdates.PendingAppleUpdatesColumns(), pendingappleupdates.PendingAppleUpdatesGenerate))
	log.Infof("Registered macadmins table: pending_apple_updates")

	server.RegisterPlugin(table.NewPlugin("macadmins_unified_log", unifiedlog.UnifiedLogColumns(), unifiedlog.UnifiedLogGenerate))
	log.Infof("Registered macadmins table: macadmins_unified_log")

	server.RegisterPlugin(table.NewPlugin("macos_rsr", macosrsr.MacOSRsrColumns(), macosrsr.MacOSRsrGenerate))
	log.Infof("Registered macadmins table: macos_rsr")

	server.RegisterPlugin(table.NewPlugin("sofa_security_release_info", sofa.SofaSecurityReleaseInfoColumns(), func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		return sofa.SofaSecurityReleaseInfoGenerate(ctx, queryContext, socketPath, sofaOpts...)
	}))
	log.Infof("Registered macadmins table: sofa_security_release_info")

	server.RegisterPlugin(table.NewPlugin("sofa_unpatched_cves", sofa.SofaUnpatchedCVEsColumns(), func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		return sofa.SofaUnpatchedCVEsGenerate(ctx, queryContext, socketPath, sofaOpts...)
	}))
	log.Infof("Registered macadmins table: sofa_unpatched_cves")

	server.RegisterPlugin(table.NewPlugin("authdb", authdb.AuthDBColumns(), authdb.AuthDBGenerate))
	log.Infof("Registered macadmins table: authdb")

	server.RegisterPlugin(table.NewPlugin(
		"wifi_network",
		wifi_network.WifiNetworkColumns(),
		func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
			return wifi_network.WifiNetworkGenerate(ctx, queryContext, socketPath)
		},
	))
	log.Infof("Registered macadmins table: wifi_network")

	server.RegisterPlugin(table.NewPlugin("alt_system_info", alt_system_info.AltSystemInfoColumns(),
		func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
			return alt_system_info.AltSystemInfoGenerate(ctx, queryContext, socketPath)
		},
	))
	log.Infof("Registered macadmins table: alt_system_info")
}
