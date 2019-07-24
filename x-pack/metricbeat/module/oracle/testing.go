// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package oracle

import (
	"fmt"
	"os"

	"gopkg.in/goracle.v2"
)

// GetOracleConnectionDetails return a valid SID to use for testing
func GetOracleConnectionDetails() string {
	params := goracle.ConnectionParams{
		SID:      fmt.Sprintf("%s:%s/%s", GetOracleEnvHost(), GetOracleEnvPort(), GetOracleEnvServiceName()),
		Username: GetOracleEnvUsername(),
		Password: GetOracleEnvPassword(),
		IsSysDBA: true,
	}

	return params.StringWithPassword()
}

// GetOracleEnvHost returns the hostname to use with Oracle testing server or the value of the environment variable ORACLE_HOST if not empty
func GetOracleEnvHost() string {
	host := os.Getenv("ORACLE_HOST")

	if len(host) == 0 {
		host = "localhost"
	}
	return host
}

// GetOracleEnvPort returns the port to use with Oracle testing server or the value of the environment variable ORACLE_PORT if not empty
func GetOracleEnvPort() string {
	port := os.Getenv("ORACLE_PORT")

	if len(port) == 0 {
		port = "1521"
	}
	return port
}

// GetOracleEnvServiceName returns the service name to use with Oracle testing server or the value of the environment variable ORACLE_SERVICE_NAME if not empty
func GetOracleEnvServiceName() string {
	serviceName := os.Getenv("ORACLE_SERVICE_NAME")

	if len(serviceName) == 0 {
		serviceName = "ORCLPDB1.localdomain"
	}
	return serviceName
}

// GetOracleEnvUsername returns the username to use with Oracle testing server or the value of the environment variable ORACLE_USERNAME if not empty
func GetOracleEnvUsername() string {
	username := os.Getenv("ORACLE_USERNAME")

	if len(username) == 0 {
		username = "sys"
	}
	return username
}

// GetOracleEnvUsername returns the port of the Oracle server or the value of the environment variable ORACLE_PASSWORD if not empty
func GetOracleEnvPassword() string {
	password := os.Getenv("ORACLE_PASSWORD")

	if len(password) == 0 {
		password = "Oradoc_db1"
	}
	return password
}
