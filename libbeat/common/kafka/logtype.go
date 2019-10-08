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

package kafka

import (
	"fmt"
)

// LogType is the type of the messages coming in
type LogType string

var (
	// ActivityLogs variable used to identify the activitylogs azure metricset
	ActivityLogs LogType = "ActivityLogs"
	// AuditLogs variable used to identify the auditlogs azure metricset
	AuditLogs  LogType = "AuditLogs"
	// SigninLogs variable used to identify the signinlogs azure metricset
	SigninLogs  LogType = "SigninLogs"

	LogTypes = map[string]LogType{
		"AuditLogs":    AuditLogs,
		"ActivityLogs": ActivityLogs,
		"SigninLogs":   SigninLogs,
	}
)

// Validate that a log type is among the possible options
func (lt *LogType) Validate() error {
	if _, ok := LogTypes[string(*lt)]; !ok {
		return fmt.Errorf("unknown/unsupported kafka vesion '%v'", *lt)
	}

	return nil
}

// Unpack the log type version
func (lt *LogType) Unpack(s string) error {
	tmp := LogType(s)
	if err := tmp.Validate(); err != nil {
		return err
	}
	*lt = tmp
	return nil
}

// Get the log type
func (lt LogType) Get() (LogType, bool) {
	kv, ok := LogTypes[string(lt)]
	return kv, ok
}
