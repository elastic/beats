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

package aucoalesce

import (
	"math"
	"os/user"
	"strings"
	"sync"
	"time"
)

const cacheTimeout = time.Minute

var (
	userLookup  = NewUserCache(cacheTimeout)
	groupLookup = NewGroupCache(cacheTimeout)
)

type stringItem struct {
	timeout time.Time
	value   string
}

func (i *stringItem) isExpired() bool {
	return time.Now().After(i.timeout)
}

// UserCache is a cache of UID to username.
type UserCache struct {
	expiration time.Duration
	data       map[string]stringItem
	mutex      sync.Mutex
}

// NewUserCache returns a new UserCache. UserCache is thread-safe.
func NewUserCache(expiration time.Duration) *UserCache {
	return &UserCache{
		expiration: expiration,
		data: map[string]stringItem{
			"0": {timeout: time.Unix(math.MaxInt64, 0), value: "root"},
		},
	}
}

// LookupUID looks up a UID and returns the username associated with it. If
// no username could be found an empty string is returned. The value will be
// cached for a minute. This requires cgo on Linux.
func (c *UserCache) LookupUID(uid string) string {
	if uid == "" || uid == "unset" {
		return ""
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if item, found := c.data[uid]; found && !item.isExpired() {
		return item.value
	}

	// Cache the value (even on error).
	user, err := user.LookupId(uid)
	if err != nil {
		c.data[uid] = stringItem{timeout: time.Now().Add(c.expiration), value: ""}
		return ""
	}

	c.data[uid] = stringItem{timeout: time.Now().Add(c.expiration), value: user.Username}
	return user.Username
}

// GroupCache is a cache of GID to group name.
type GroupCache struct {
	expiration time.Duration
	data       map[string]stringItem
	mutex      sync.Mutex
}

// NewGroupCache returns a new GroupCache. GroupCache is thread-safe.
func NewGroupCache(expiration time.Duration) *GroupCache {
	return &GroupCache{
		expiration: expiration,
		data: map[string]stringItem{
			"0": {timeout: time.Unix(math.MaxInt64, 0), value: "root"},
		},
	}
}

// LookupGID looks up a GID and returns the group associated with it. If
// no group could be found an empty string is returned. The value will be
// cached for a minute. This requires cgo on Linux.
func (c *GroupCache) LookupGID(gid string) string {
	if gid == "" || gid == "unset" {
		return ""
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if item, found := c.data[gid]; found && !item.isExpired() {
		return item.value
	}

	// Cache the value (even on error).
	group, err := user.LookupGroupId(gid)
	if err != nil {
		c.data[gid] = stringItem{timeout: time.Now().Add(c.expiration), value: ""}
		return ""
	}

	c.data[gid] = stringItem{timeout: time.Now().Add(c.expiration), value: group.Name}
	return group.Name
}

// ResolveIDs translates all uid and gid values to their associated names.
// Prior to Go 1.9 this requires cgo on Linux. UID and GID values are cached
// for 60 seconds from the time they are read.
func ResolveIDs(event *Event) {
	ResolveIDsFromCaches(event, userLookup, groupLookup)
}

// ResolveIDsFromCaches translates all uid and gid values to their associated
// names using the provided caches. Prior to Go 1.9 this requires cgo on Linux.
func ResolveIDsFromCaches(event *Event, users *UserCache, groups *GroupCache) {
	// Actor
	if v := users.LookupUID(event.Summary.Actor.Primary); v != "" {
		event.Summary.Actor.Primary = v
	}
	if v := users.LookupUID(event.Summary.Actor.Secondary); v != "" {
		event.Summary.Actor.Secondary = v
	}

	// User
	names := map[string]string{}
	for key, id := range event.User.IDs {
		if strings.HasSuffix(key, "uid") {
			if v := users.LookupUID(id); v != "" {
				names[key] = v
			}
		} else if strings.HasSuffix(key, "gid") {
			if v := groups.LookupGID(id); v != "" {
				names[key] = v
			}
		}
	}
	if len(names) > 0 {
		event.User.Names = names
	}

	// File owner/group
	if event.File != nil {
		if event.File.UID != "" {
			event.File.Owner = users.LookupUID(event.File.UID)
		}
		if event.File.GID != "" {
			event.File.Group = groups.LookupGID(event.File.GID)
		}
	}
}
