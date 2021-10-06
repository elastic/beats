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
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLicenseUnmarshal(t *testing.T) {
	tests := map[string]struct {
		lic     string
		want    License
		failing bool
	}{
		"basic license": {
			lic:  `{"uid": "1234", "type": "basic", "status": "active"}`,
			want: License{UUID: "1234", Type: Basic, Status: Active},
		},
		"active enterprise": {
			lic:  `{"uid": "1234", "type": "enterprise", "status": "active"}`,
			want: License{UUID: "1234", Type: Enterprise, Status: Active},
		},
		"expired enterprise": {
			lic:  `{"uid": "1234", "type": "enterprise", "status": "expired"}`,
			want: License{UUID: "1234", Type: Enterprise, Status: Expired},
		},
		"trial": {
			// 2018-09-27 15:06:21.728 +0000 UTC
			lic:  `{"uid": "1234", "type": "trial", "status": "active", "expiry_date_in_millis": 1538060781728}`,
			want: License{UUID: "1234", Type: Trial, Status: Active, ExpiryDate: time.Date(2018, 9, 27, 15, 6, 21, 728000000, time.UTC)},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var got License
			if err := json.Unmarshal([]byte(test.lic), &got); err != nil {
				if !test.failing {
					t.Fatal(err)
				}
				return
			}

			if test.failing {
				t.Fatal("expected license to be invalid")
			}

			assert.Equal(t, test.want, got)
		})
	}
}

func TestIsExpired(t *testing.T) {
	expired := true

	tests := map[string]struct {
		license License
		want    bool
	}{
		"trial is expired": {
			license: License{Type: Trial, ExpiryDate: time.Now().Add(-2 * time.Hour)},
			want:    expired,
		},
		"trial is not expired": {
			license: License{Type: Trial, ExpiryDate: time.Now().Add(2 * time.Minute)},
			want:    !expired,
		},
		"license is not on trial": {
			license: License{Type: Basic, ExpiryDate: time.Now().Add(2 * time.Minute)},
			want:    !expired,
		},
		"state is expired": {
			license: License{Type: Enterprise, Status: Expired},
			want:    expired,
		},
		"active enterprise license": {
			license: License{Type: Enterprise, Status: Active},
			want:    !expired,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.want, IsExpired(test.license))
		})
	}
}
