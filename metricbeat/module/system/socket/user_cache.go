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

package socket

import (
	"os/user"
	"strconv"
)

// UserCache is a cache of UID to username.
type UserCache map[int]string

// NewUserCache returns a new UserCache.
func NewUserCache() UserCache {
	return map[int]string{0: "root"}
}

// LookupUID looks up a UID and returns the username associated with it. If
// no username could be found an empty string is returned. The value will be
// cached forever.
func (c UserCache) LookupUID(uid int) string {
	if username, found := c[uid]; found {
		return username
	}

	// Cache the value (even on error).
	username, err := user.LookupId(strconv.Itoa(uid))
	if err != nil {
		c[uid] = ""
		return ""
	}
	c[uid] = username.Name
	return username.Name
}
