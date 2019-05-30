// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package oracle

import "os"

// GetOracleEnvHost returns the hostname of the Oracle server
func GetOracleEnvHost() string {
	host := os.Getenv("ORACLE_HOST")

	if len(host) == 0 {
		host = "localhost"
	}
	return host
}

// GetOracleEnvPort returns the port of the Oracle server
func GetOracleEnvPort() string {
	port := os.Getenv("ORACLE_PORT")

	if len(port) == 0 {
		port = "1521"
	}
	return port
}
