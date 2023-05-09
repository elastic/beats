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
	"fmt"
	"time"
)

// License represents the license of this beat, the license is fetched and returned from
// the elasticsearch cluster.
//
// The x-pack endpoint returns the following JSON response.
//
//	{
//	  "license" : {
//	    "status" : "active",
//	    "uid" : "cbff45e7-c553-41f7-ae4f-9205eabd80xx",
//	    "type" : "trial",
//	    "issue_date" : "2018-10-20T22:05:12.332Z",
//	    "issue_date_in_millis" : 1540073112332,
//	    "expiry_date" : "2018-11-19T22:05:12.332Z",
//	    "expiry_date_in_millis" : 1542665112332,
//	    "max_nodes" : 1000,
//	    "issued_to" : "test",
//	    "issuer" : "elasticsearch",
//	    "start_date_in_millis" : -1
//	  }
//	}
//
// Definition:
// type is the installed license.
// mode is the license in operation. (effective license)
// status is the type installed is active or not.
type License struct {
	UUID       string      `json:"uid"`
	Type       LicenseType `json:"type"`
	Status     State       `json:"status"`
	ExpiryDate time.Time   `json:"expiry_date_in_millis,omitempty"`
}

func (l *License) UnmarshalJSON(b []byte) error {
	document := struct {
		UUID       string      `json:"uid"`
		Type       LicenseType `json:"type"`
		Status     State       `json:"status"`
		ExpiryDate int64       `json:"expiry_date_in_millis,omitempty"`
	}{}

	if err := json.Unmarshal(b, &document); err != nil {
		return err
	}

	var expiryTime time.Time
	if document.ExpiryDate != 0 {
		expiryTime = time.Unix(0, int64(time.Millisecond)*int64(document.ExpiryDate)).UTC()
	} else if document.Type == Trial {
		return fmt.Errorf("missing expiry_date_in_millis on trial license")
	}

	*l = License{
		UUID:       document.UUID,
		Type:       document.Type,
		Status:     document.Status,
		ExpiryDate: expiryTime,
	}
	return nil
}

// Cover returns true if the provided license is included in the range of license.
//
// Basic -> match basic, gold and platinum
// gold -> match gold and platinum
// platinum -> match  platinum only
func (l *License) Cover(license LicenseType) bool {
	if l.Type >= license {
		return true
	}
	return false
}

// IsExpired checks if the installed Elasticsearch license is still valid.
func IsExpired(l License) bool {
	return l.Status == Expired || (l.Type == Trial && time.Now().After(l.ExpiryDate))
}

// EqualTo returns true if the two license are the same, we compare license to reduce the number
// message send to the watchers.
func (l *License) EqualTo(other *License) bool {
	return l.UUID == other.UUID &&
		l.Type == other.Type &&
		l.Status == other.Status
}
