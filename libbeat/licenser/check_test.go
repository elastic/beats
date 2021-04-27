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

package licenser

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/logp"
)

func TestCheckLicense(t *testing.T) {
	t.Run("Trial", testCheckTrial)
	t.Run("Cover", testCheckLicenseCover)
	t.Run("Validate", testValidate)
}

func testCheckTrial(t *testing.T) {
	log := logp.NewLogger("")

	t.Run("valid trial license", func(t *testing.T) {
		l := License{
			Type:        Trial,
			TrialExpiry: expiryTime(time.Now().Add(1 * time.Hour)),
		}
		assert.True(t, CheckTrial(log, l))
	})

	t.Run("expired trial license", func(t *testing.T) {
		l := License{
			Type:        Trial,
			TrialExpiry: expiryTime(time.Now().Add(-1 * time.Hour)),
		}
		assert.False(t, CheckTrial(log, l))
	})

	t.Run("other license", func(t *testing.T) {
		l := License{Type: Basic}
		assert.False(t, CheckTrial(log, l))
	})
}

func testCheckLicenseCover(t *testing.T) {
	log := logp.NewLogger("")
	lt := []LicenseType{Basic, Gold, Platinum}
	for _, license := range lt {
		fn := CheckLicenseCover(license)

		t.Run("active", func(t *testing.T) {
			l := License{Type: license, Status: Active}
			assert.True(t, fn(log, l))
		})

		t.Run("inactive", func(t *testing.T) {
			l := License{Type: license, Status: Inactive}
			assert.False(t, fn(log, l))
		})
	}
}

func testValidate(t *testing.T) {
	l := License{Type: Basic, Status: Active}
	t.Run("when one of the check is valid", func(t *testing.T) {
		valid := Validate(logp.NewLogger(""), l, CheckLicenseCover(Platinum), CheckLicenseCover(Basic))
		assert.True(t, valid)
	})

	t.Run("when no check is valid", func(t *testing.T) {
		valid := Validate(logp.NewLogger(""), l, CheckLicenseCover(Platinum), CheckLicenseCover(Gold))
		assert.False(t, valid)
	})
}
