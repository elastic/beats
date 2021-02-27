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

package sys

import "golang.org/x/sys/windows"

// PopulateAccount lookups the account name and type associated with a SID.
// The account name, domain, and type are added to the given sid.
func PopulateAccount(sid *SID) error {
	if sid == nil || sid.Identifier == "" {
		return nil
	}

	s, err := windows.StringToSid(sid.Identifier)
	if err != nil {
		return err
	}

	account, domain, accType, err := s.LookupAccount("")
	if err != nil {
		return err
	}

	sid.Name = account
	sid.Domain = domain
	sid.Type = SIDType(accType)
	return nil
}
