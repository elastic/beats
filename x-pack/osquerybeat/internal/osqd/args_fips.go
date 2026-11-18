// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build requirefips

package osqd

import (
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/elastic/beats/v7/pkg/logp"
)

// enforceFIPSFlags overwrites flags that must not change in a FIPS build and
// adds fipsDisabledTables to the disabled tables. It logs each user setting it
// replaces. It modifies flags in place and returns it, for use in a single
// expression.
func enforceFIPSFlags(flags Flags, log *logp.Logger) Flags {
	for k, want := range fipsFlags {
		// Compare as text, so that the string "true" and the bool true are equal.
		if got, ok := flags[k]; ok && fmt.Sprint(got) != fmt.Sprint(want) {
			log.Warnf("Ignoring osqueryd flag %s=%v, FIPS builds always use %v", k, got, want)
		}
		flags[k] = want
	}

	// osqueryd gives disable_tables priority over enable_tables, so the
	// enable_tables flag can stay as the user set it. flags can hold no
	// disable_tables string, for example when the user enables both default
	// tables, and GetString then returns an empty list.
	flags[flagDisableTables] = mergeTables(flags.GetString(flagDisableTables), fipsDisabledTables)

	return flags
}

// EnforceFIPSOptions returns a copy of options, for the options section of the
// osquery config, with the FIPS flags set to the values that FIPS builds
// enforce.
//
// osqueryd applies config options over command-line flags for every flag that
// is not CLI-only, so a policy could otherwise turn disable_distributed and
// yara_sigurl_authenticate back on. The CLI-only flags are set too, in case
// that changes in a later osquery version. osqueryd ignores them with a
// warning.
//
// It does not log. The same policy options also become command-line flags,
// and enforceFIPSFlags logs each value it replaces.
func EnforceFIPSOptions(options map[string]any) map[string]any {
	enforced := maps.Clone(options)
	if enforced == nil {
		enforced = make(map[string]any, len(fipsFlags))
	}
	maps.Copy(enforced, fipsFlags)
	return enforced
}

// fipsFlags are the osqueryd flags that FIPS builds always pass, whatever the
// policy asks for. They turn off the features that can use osqueryd's bundled
// OpenSSL, which has no FIPS module, for TLS.
//
// In osquery 5.24.0 only disable_distributed and yara_sigurl_authenticate can
// be changed through the options in the osquery config. The other three are
// CLI-only.
var fipsFlags = Flags{
	"disable_distributed":      true,
	"disable_enrollment":       true,
	"disable_carver":           true,
	"aws_enforce_fips":         true,
	"yara_sigurl_authenticate": false,
}

// fipsDisabledTables are the tables that FIPS builds always disable, whatever
// the policy asks for. enable_tables cannot re-enable them.
//
// The first group can make TLS or other remote calls. yara is not listed
// because it is an alias of yara_file, and disabling yara_file removes the
// alias. The rpm tables are disabled because librpm verifies package headers
// with the bundled OpenSSL.
var fipsDisabledTables = []string{
	"carves",
	"curl",
	"curl_certificate",
	"prometheus_metrics",
	"ec2_instance_metadata",
	"ec2_instance_tags",
	"azure_instance_metadata",
	"azure_instance_tags",
	"ycloud_instance_metadata",
	"yara_file",
	"yara_process",
	"yara_events",

	"rpm_packages",
	"rpm_package_files",
}

// mergeTables returns the sorted, comma separated union of the comma
// separated list and the names.
func mergeTables(list string, names []string) string {
	set := make(map[string]struct{}, len(names))
	for _, name := range names {
		set[name] = struct{}{}
	}
	for name := range strings.SplitSeq(list, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			set[name] = struct{}{}
		}
	}

	merged := make([]string, 0, len(set))
	for name := range set {
		merged = append(merged, name)
	}
	sort.Strings(merged)
	return strings.Join(merged, ",")
}
