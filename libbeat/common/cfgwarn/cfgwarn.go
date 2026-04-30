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
)

// Beta returns a string suitable to log beta feature.
func Beta(format string, v ...interface{}) string {
	return fmt.Sprintf("BETA: "+format, v...)
}

// Deprecate returns a deprecation message.
// The version string contains the version when the future will be removed.
// If version is empty, the message  will not mention the removal of the feature.
func Deprecate(version string, format string, v ...interface{}) string {
	var postfix string
	if version != "" {
		postfix = fmt.Sprintf(" Will be removed in version: %s", version)
	}
	return fmt.Sprintf("DEPRECATED: "+format+postfix, v...)
}

// Experimental returns a "usage of an experimental feature" message.
func Experimental(format string, v ...interface{}) string {
	return fmt.Sprintf("EXPERIMENTAL: "+format, v...)
}

// Preview returns a "usage of a preview feature" message.
func Preview(format string, v ...interface{}) string {
	return fmt.Sprintf("PREVIEW: "+format, v...)
}
