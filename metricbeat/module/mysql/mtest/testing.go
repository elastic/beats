// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package mtest

import (
	"github.com/go-sql-driver/mysql"

	"github.com/elastic/beats/libbeat/tests/compose"
)

// Runner is a compose test runner for mysql
var Runner = compose.TestRunner{
	Service:  "mysql",
	Parallel: true,
}

// GetDSN returns the MySQL server DSN to use for testing.
func GetDSN(host string) string {
	c := &mysql.Config{
		Net:    "tcp",
		Addr:   host,
		User:   "root",
		Passwd: "test",

		// Required if password is set and FormatDSN() is used
		// since clients for MySQL 8.0
		AllowNativePasswords: true,
	}
	return c.FormatDSN()
}

// GetConfig returns the configuration for a mysql module
func GetConfig(metricset, host string, raw bool) map[string]interface{} {
	return map[string]interface{}{
		"module":     "mysql",
		"metricsets": []string{metricset},
		"hosts":      []string{GetDSN(host)},
		"raw":        raw,
	}
}
