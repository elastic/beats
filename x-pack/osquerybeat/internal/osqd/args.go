// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package osqd

import (
	"fmt"
	"reflect"
	"strings"
)

const (
	osqueryPid      = "osquery.pid"
	osqueryDb       = "osquery.db"
	osqueryAutoload = "osquery.autoload"
	osqueryFlagfile = "osquery.flags"

	defaultExtensionsInterval = 3
	defaultExtensionsTimeout  = 10
)

type Args []string
type Flags map[string]interface{}

func (f Flags) GetString(key string) string {
	if f == nil {
		return ""
	}
	if v, ok := f[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func FlagsAreSame(flags1, flags2 Flags) bool {
	return reflect.DeepEqual(flags1, flags2)
}

// Some flags combinations to enable events collection and audits
// // Enable events collection
// "--disable_events=false",
// // Begin: enable process events audit
// "--disable_audit=false",
// "--audit_allow_config=true",
// "--audit_persist=true",
// "--audit_allow_process_events=true",
// // End: enable process events audit

// // Begin: enable sockets audit
// "--audit_allow_sockets=true",
// "--audit_allow_unix=true", // Allow domain sockets audit
// // End: enable sockets audit

var protectedFlags = Flags{
	"force":            true,
	"disable_watchdog": true,
	"utc":              true,

	// Setting this value to 1 will auto-clear events whenever a SELECT is performed against the table, reducing all impact of the buffer.
	"events_expiry": 1,

	// Extensions socket path
	"extensions_socket":   "",
	"extensions_interval": defaultExtensionsInterval,
	"extensions_timeout":  defaultExtensionsTimeout,

	// Path dependendent keys
	"pidfile":             osqueryPid,
	"database_path":       osqueryDb,
	"extensions_autoload": osqueryAutoload,
	"flagfile":            osqueryFlagfile,

	// Plugins
	"config_plugin": "",
	"logger_plugin": "",

	// The delimiter for a full query name that is concatenated as "pack_" + {{pack name}} + "_" + {{query name}} by default
	"pack_delimiter": "_",

	// This enforces the batch format for differential results
	// https://osquery.readthedocs.io/en/stable/deployment/logging
	"logger_event_type": false,

	// Refresh config every 60 seconds
	// The previous setting was 10 seconds which is unnecessary frequent.
	// Osquery does not expect that frequent policy/configuration changes
	// and can tolerate non real-time configuration change application.
	"config_refresh": 60,

	// certificates to use for curl table for example
	"tls_server_certs": "certs.pem",

	// Augeas lenses are bundled with osquery distributions
	"augeas_lenses": "lenses",
}

func init() {
	// Append platform specific flags
	plArgs := platformArgs()
	for k, v := range plArgs {
		protectedFlags[k] = v
	}
}

func convertToArgs(flags Flags) Args {
	if flags == nil {
		return nil
	}

	sz := len(flags)
	args := make([]string, 0, sz)
	for k, v := range flags {
		sval := fmt.Sprint(v)
		// Appending args, skipping the values that contain space
		if !strings.ContainsRune(sval, ' ') {
			s := fmt.Sprint("--", k, "=", v)
			args = append(args, s)
		}
	}
	return args
}
