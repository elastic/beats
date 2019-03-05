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

package perfmon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPdhErrno checks that PdhError provides the correct message for known
// PDH errors and also falls back to Windows error messages for non-PDH errors.
func TestPdhErrno_Error(t *testing.T) {
	assert.Contains(t, PdhErrno(PDH_CSTATUS_BAD_COUNTERNAME).Error(), "Unable to parse the counter path.")
	assert.Contains(t, PdhErrno(15).Error(), "The system cannot find the drive specified.")
}
