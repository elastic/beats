// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package testing

import "os"

// GetConfig returns the required configuration options for testing a MSSQL
// metricset.
func GetConfig(metricSets ...string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "mssql",
		"metricsets": metricSets,
		"hosts":      []string{EnvOr("MSSQL_HOST", "localhost")},
		"username":   EnvOr("MSSQL_USER", "SA"),
		"password":   EnvOr("MSSQL_PASSWORD", ""),
	}
}

// EnvOr returns the value of the specified environment variable if it is
// non-empty. Otherwise it return def.
func EnvOr(name, def string) string {
	s := os.Getenv(name)
	if s == "" {
		return def
	}
	return s
}
