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

package sysinit

import (
	"path/filepath"

	"github.com/menderesk/beats/v7/metricbeat/mb"
)

// Module represents the system/linux module
type Module struct {
	mb.BaseModule
	HostFS        string
	UserSetHostFS bool
}

// ResolveHostFS returns a full path based on a user-suppled path, and impliments the Resolver interface
// This is mostly to prevent any chance that other metricsets will develop their own way of
// using a user-suppied hostfs flag. We try to do all the logic in one place.
func (m Module) ResolveHostFS(path string) string {
	return filepath.Join(m.HostFS, path)
}

func (m Module) IsSet() bool {
	return m.UserSetHostFS
}

func (m Module) Join(path ...string) string {
	fullpath := append([]string{m.HostFS}, path...)
	return filepath.Join(fullpath...)

}
