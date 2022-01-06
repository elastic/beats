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

package cfgwarn

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/elastic/beats/v7/libbeat/logp"
)

const selector = "cfgwarn"

// Beta logs the usage of an beta feature.
func Beta(format string, v ...interface{}) {
	logp.NewLogger(selector, zap.AddCallerSkip(1)).Warnf("BETA: "+format, v...)
}

// Deprecate logs a deprecation message.
// The version string contains the version when the future will be removed.
// If version is empty, the message  will not mention the removal of the feature.
func Deprecate(version string, format string, v ...interface{}) {
	var postfix string
	if version != "" {
		postfix = fmt.Sprintf(" Will be removed in version: %s", version)
	}
	logp.NewLogger(selector, zap.AddCallerSkip(1)).Warnf("DEPRECATED: "+format+postfix, v...)
}

// Experimental logs the usage of an experimental feature.
func Experimental(format string, v ...interface{}) {
	logp.NewLogger(selector, zap.AddCallerSkip(1)).Warnf("EXPERIMENTAL: "+format, v...)
}
