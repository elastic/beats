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

package useragent

import (
	"fmt"
	"runtime"

	"github.com/elastic/beats/libbeat/version"
)

// UserAgent takes the capitalized name of the current beat and returns
// an RFC compliant user agent string for that beat.
func UserAgent(beatNameCapitalized string) string {
	return fmt.Sprintf("Elastic-%s/%s (%s; %s; %s; %s)",
		beatNameCapitalized,
		version.GetDefaultVersion(), runtime.GOOS, runtime.GOARCH,
		version.Commit(), version.BuildTime())
}
